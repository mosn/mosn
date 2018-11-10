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

package http

import (
	"context"
	"sync"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/proxy"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/rcrowley/go-metrics"
	"time"
)

const defaultMaxConn = 512
const defaultIdleTimeout = time.Second * 60

func init() {
	proxy.RegisterNewPoolFactory(protocol.HTTP1, NewConnPool)
	types.RegisterConnPoolFactory(protocol.HTTP1, true)

}

// types.ConnectionPool
type connPool struct {
	MaxConn int

	host types.Host

	clientMux   sync.Mutex
	clients     []*activeClient // available clients
	clientCount int             // total clients
}

func NewConnPool(host types.Host) types.ConnectionPool {
	return &connPool{
		host: host,
	}
}

func (p *connPool) Protocol() types.Protocol {
	return protocol.HTTP1
}

//由 PROXY 调用
func (p *connPool) NewStream(ctx context.Context, streamID string, responseDecoder types.StreamReceiver,
	cb types.PoolEventListener) types.Cancellable {

	c := p.getAvailableClient(ctx)

	if c == nil {
		cb.OnFailure(streamID, types.ConnectionFailure, nil)
		return nil
	}

	if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		cb.OnFailure(streamID, types.Overflow, nil)
		p.host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
	} else {
		p.host.HostStats().UpstreamRequestTotal.Inc(1)
		p.host.HostStats().UpstreamRequestActive.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
		p.host.ClusterInfo().ResourceManager().Requests().Increase()

		streamEncoder := c.codecClient.NewStream(ctx, streamID, responseDecoder)
		cb.OnReady(streamID, streamEncoder, p.host)
	}

	return nil
}

func (p *connPool) getAvailableClient(ctx context.Context) *activeClient {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	n := len(p.clients)
	// no available client
	if n == 0 {
		maxConns := p.MaxConn
		if maxConns <= 0 {
			maxConns = defaultMaxConn
		}
		if p.clientCount < maxConns {
			p.clientCount++
			return newActiveClient(ctx, p)
		}
	} else {
		n--
		c := p.clients[n]
		p.clients[n] = nil
		p.clients = p.clients[:n]
		return c
	}

	return nil
}

func (p *connPool) Close() {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	for _, c := range p.clients {
		c.codecClient.Close()
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

		p.clientCount--

		// check if closed connection is available
		p.clientMux.Lock()
		defer p.clientMux.Unlock()

		for i, c := range p.clients {
			if c == client {
				p.clients[i] = nil
				p.clients = append(p.clients[:i], p.clients[i+1:]...)
			}
		}
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

	// return to pool
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	p.clients = append(p.clients, client)
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
	p.host.HostStats().UpstreamConnectionCloseNotify.Inc(1)
	p.host.ClusterInfo().Stats().UpstreamConnectionCloseNotify.Inc(1)

	// http/1.x should not enter this branch
	//
	//if p.primaryClient == client {
	//	p.movePrimaryToDraining()
	//}
}

func (p *connPool) createCodecClient(context context.Context, connData types.CreateConnectionData) str.CodecClient {
	return str.NewCodecClient(context, protocol.HTTP1, connData.Connection, connData.HostInfo)
}

// stream.CodecClientCallbacks
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	pool               *connPool
	codecClient        str.CodecClient
	host               types.CreateConnectionData
	totalStream        uint64
	closeWithActiveReq bool
}

func newActiveClient(context context.Context, pool *connPool) *activeClient {
	ac := &activeClient{
		pool: pool,
	}

	data := pool.host.CreateConnection(context)
	codecClient := pool.createCodecClient(context, data)
	codecClient.AddConnectionCallbacks(ac)
	codecClient.SetCodecClientCallbacks(ac)
	codecClient.SetCodecConnectionCallbacks(ac)

	ac.codecClient = codecClient
	ac.host = data

	if err := ac.host.Connection.Connect(true); err != nil {
		return nil
	}

	pool.host.HostStats().UpstreamConnectionTotal.Inc(1)
	pool.host.HostStats().UpstreamConnectionActive.Inc(1)
	//pool.host.HostStats().UpstreamConnectionTotalHTTP1.Inc(1)
	pool.host.ClusterInfo().Stats().UpstreamConnectionTotal.Inc(1)
	pool.host.ClusterInfo().Stats().UpstreamConnectionActive.Inc(1)
	//pool.host.ClusterInfo().Stats().UpstreamConnectionTotalHTTP1.Inc(1)

	// bytes total adds all connections data together, but buffered data not
	codecClient.SetConnectionStats(&types.ConnectionStats{
		ReadTotal:     pool.host.ClusterInfo().Stats().UpstreamBytesReadTotal,
		ReadBuffered:  metrics.NewGauge(),
		WriteTotal:    pool.host.ClusterInfo().Stats().UpstreamBytesWriteTotal,
		WriteBuffered: metrics.NewGauge(),
	})

	return ac
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
