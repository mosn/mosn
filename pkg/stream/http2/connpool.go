/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http2

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/mtls"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/proxy"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	metrics "github.com/rcrowley/go-metrics"
	"golang.org/x/net/http2"
)

const (
	// H2 conn key in context
	H2ConnKey = "h2_conn"
)

func init() {
	proxy.RegisterNewPoolFactory(protocol.HTTP2, NewConnPool)
	types.RegisterConnPoolFactory(protocol.HTTP2, true)
}

// types.ConnectionPool
type connPool struct {
	activeClients map[string][]*activeClient // key is host:port
	mux           sync.RWMutex
	host          types.Host
}

func NewConnPool(host types.Host) types.ConnectionPool {
	return &connPool{
		host:          host,
		activeClients: make(map[string][]*activeClient),
	}
}

func (p *connPool) Protocol() types.Protocol {
	return protocol.HTTP2
}

func (p *connPool) Host() types.Host {
	return p.host
}

func (p *connPool) InitActiveClient(context context.Context) error {
	return nil
}

//由 PROXY 调用
func (p *connPool) NewStream(context context.Context, streamID string, responseDecoder types.StreamReceiver,
	cb types.PoolEventListener) types.Cancellable {

	ac := p.getOrInitActiveClient(context, p.host.AddressString())

	if ac == nil {
		cb.OnFailure(streamID, types.ConnectionFailure, nil)
		return nil
	}

	if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		cb.OnFailure(streamID, types.Overflow, nil)
		p.host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
	} else {
		atomic.AddUint64(&ac.totalStream, 1)
		p.host.HostStats().UpstreamRequestTotal.Inc(1)
		p.host.HostStats().UpstreamRequestActive.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
		p.host.ClusterInfo().ResourceManager().Requests().Increase()
		streamEncoder := ac.codecClient.NewStream(context, streamID, responseDecoder)
		cb.OnReady(streamID, streamEncoder, p.host)
	}

	return nil
}

func (p *connPool) Close() {
	p.mux.Lock()
	defer p.mux.Unlock()

	for _, acs := range p.activeClients {
		for _, ac := range acs {
			ac.codecClient.Close()
		}
	}
}

func (p *connPool) onConnectionEvent(client *activeClient, event types.ConnectionEvent) {
	if event.IsClose() {

		if client.closeWithActiveReq {
			if event == types.LocalClose {
				p.host.HostStats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
				p.host.ClusterInfo().Stats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
			} else if event == types.RemoteClose {
				p.host.HostStats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
				p.host.ClusterInfo().Stats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
			}
		}

		host := client.host.HostInfo.AddressString()

		p.mux.Lock()
		defer p.mux.Unlock()

		delete(p.activeClients, host)
	} else if event == types.ConnectTimeout {
		p.host.HostStats().UpstreamRequestTimeout.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestTimeout.Inc(1)
		client.codecClient.Close()
	} else if event == types.ConnectFailed {
		p.host.HostStats().UpstreamConnectionConFail.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamConnectionConFail.Inc(1)
	}
}

func (p *connPool) onStreamDestroy(client *activeClient) {
	p.host.HostStats().UpstreamRequestActive.Dec(1)
	p.host.ClusterInfo().Stats().UpstreamRequestActive.Dec(1)
	p.host.ClusterInfo().ResourceManager().Requests().Decrease()
}

func (p *connPool) onStreamReset(client *activeClient, reason types.StreamResetReason) {
	if reason == types.StreamConnectionTermination || reason == types.StreamConnectionFailed {
		p.host.HostStats().UpstreamRequestFailureEject.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestFailureEject.Inc(1)
		client.closeWithActiveReq = true
	} else if reason == types.StreamLocalReset {
		p.host.HostStats().UpstreamRequestLocalReset.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestLocalReset.Inc(1)
	} else if reason == types.StreamRemoteReset {
		p.host.HostStats().UpstreamRequestRemoteReset.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestRemoteReset.Inc(1)
	}
}

func (p *connPool) onGoAway(client *activeClient) {
	if !atomic.CompareAndSwapUint32(&client.isGoneAway, 0, 1) {
		return
	}

	p.host.HostStats().UpstreamConnectionCloseNotify.Inc(1)
	p.host.ClusterInfo().Stats().UpstreamConnectionCloseNotify.Inc(1)
	p.mux.Lock()
	defer p.mux.Unlock()

	host := client.host.HostInfo.AddressString()
	delete(p.activeClients, host)
}

func (p *connPool) createCodecClient(context context.Context, connData types.CreateConnectionData) str.CodecClient {
	return str.NewCodecClient(context, protocol.HTTP2, connData.Connection, connData.HostInfo)
}

// Http2 connpool interface
func (p *connPool) getOrInitActiveClient(context context.Context, addr string) *activeClient {
	p.mux.Lock()
	defer p.mux.Unlock()

	for _, ac := range p.activeClients[addr] {
		if ac.CanTakeNewRequest() {

			return ac
		}
	}

	// If connection's stream id is out of bound, closed or 'go away', make a new one
	if nac := newActiveClient(context, p); nac != nil {
		p.activeClients[addr] = append(p.activeClients[addr], nac)

		return nac
	}

	return nil
}

// GetClientConn
func (p *connPool) GetClientConn(req *http.Request, addr string) (*http2.ClientConn, error) {
	// GetClientConn will not be called, do nothing
	return nil, nil
}

// MarkDead by golang net http2 impl
func (p *connPool) MarkDead(http2Conn *http2.ClientConn) {
	acsIdx := ""
	acIdx := -1
	var deadAc *activeClient

	p.mux.Lock()

	for i, acs := range p.activeClients {
		for j, ac := range acs {

			if ac.h2Conn == http2Conn {
				acsIdx = i
				acIdx = j
				deadAc = ac

				break
			}
		}
	}

	if acsIdx != "" && acIdx > -1 {
		// close connection to notify watchers
		p.activeClients[acsIdx] = append(p.activeClients[acsIdx][:acIdx],
			p.activeClients[acsIdx][acIdx+1:]...)
	}

	p.mux.Unlock()

	if deadAc != nil {
		deadAc.host.Connection.Close(types.NoFlush, types.LocalClose)
	}
}

// stream.CodecClientCallbacks
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	pool *connPool

	codecClient        str.CodecClient
	h2Conn             *http2.ClientConn
	host               types.CreateConnectionData
	totalStream        uint64
	closeWithActiveReq bool
	isGoneAway         uint32
}

func newActiveClient(ctx context.Context, pool *connPool) *activeClient {
	ac := &activeClient{
		pool: pool,
	}

	data := pool.host.CreateConnection(ctx)

	if err := data.Connection.Connect(false); err != nil {
		return nil
	}

	transport := &http2.Transport{
		ConnPool: pool,
	}

	if tlsConn, ok := data.Connection.RawConn().(*mtls.TLSConn); ok {
		if err := tlsConn.Handshake(); err != nil {
			data.Connection.Close(types.NoFlush, types.ConnectFailed)
			return nil
		}
	}

	h2Conn, err := transport.NewClientConn(data.Connection.RawConn())

	if err != nil {
		// close connection on h2 conn init failed
		data.Connection.Close(types.NoFlush, types.LocalClose)

		return nil
	}

	codecClient := pool.createCodecClient(context.WithValue(ctx, H2ConnKey, h2Conn), data)
	codecClient.AddConnectionCallbacks(ac)
	codecClient.SetCodecClientCallbacks(ac)
	codecClient.SetCodecConnectionCallbacks(ac)

	ac.host = data
	ac.h2Conn = h2Conn
	ac.codecClient = codecClient

	pool.host.HostStats().UpstreamConnectionTotal.Inc(1)
	pool.host.HostStats().UpstreamConnectionActive.Inc(1)
	//pool.host.HostStats().UpstreamConnectionTotalHTTP2.Inc(1)
	pool.host.ClusterInfo().Stats().UpstreamConnectionTotal.Inc(1)
	pool.host.ClusterInfo().Stats().UpstreamConnectionActive.Inc(1)
	//pool.host.ClusterInfo().Stats().UpstreamConnectionTotalHTTP2.Inc(1)

	// bytes total adds all connections data together, but buffered data not
	codecClient.SetConnectionStats(&types.ConnectionStats{
		ReadTotal:     pool.host.ClusterInfo().Stats().UpstreamBytesReadTotal,
		ReadBuffered:  metrics.NewGauge(),
		WriteTotal:    pool.host.ClusterInfo().Stats().UpstreamBytesWriteTotal,
		WriteBuffered: metrics.NewGauge(),
	})

	return ac
}

func (ac *activeClient) CanTakeNewRequest() bool {
	// extend this method if we need to control this in stream layer
	return ac.h2Conn.CanTakeNewRequest()
}

func (ac *activeClient) OnEvent(event types.ConnectionEvent) {
	ac.pool.onConnectionEvent(ac, event)
}

func (ac *activeClient) OnStreamDestroy() {
	ac.pool.onStreamDestroy(ac)
}

func (ac *activeClient) OnStreamReset(reason types.StreamResetReason) {
	ac.pool.onStreamReset(ac, reason)
}

func (ac *activeClient) OnGoAway() {
	ac.pool.onGoAway(ac)
}
