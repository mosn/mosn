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
	"fmt"
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

	statReport bool

	clientMux   sync.Mutex
	clients     []*activeClient // available clients
	clientCount int             // total clients
}

func NewConnPool(host types.Host) types.ConnectionPool {
	pool := &connPool{
		host: host,
	}

	if pool.statReport {
		pool.report()
	}

	return pool
}

func (p *connPool) Protocol() types.Protocol {
	return protocol.HTTP1
}

//由 PROXY 调用
func (p *connPool) NewStream(ctx context.Context, receiver types.StreamReceiver, cb types.PoolEventListener) types.Cancellable {

	c := p.getAvailableClient(ctx)

	if c == nil {
		cb.OnFailure(types.ConnectionFailure, nil)
		return nil
	}

	if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		cb.OnFailure(types.Overflow, nil)
		p.host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
	} else {
		p.host.HostStats().UpstreamRequestTotal.Inc(1)
		p.host.HostStats().UpstreamRequestActive.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
		p.host.ClusterInfo().ResourceManager().Requests().Increase()

		streamEncoder := c.codecClient.NewStream(ctx, receiver)
		cb.OnReady(streamEncoder, p.host)
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

		// check if closed connection is available
		p.clientMux.Lock()
		defer p.clientMux.Unlock()

		p.clientCount--

		for i, c := range p.clients {
			if c == client {
				p.clients[i] = nil
				p.clients = append(p.clients[:i], p.clients[i+1:]...)
				break
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
	p.clients = append(p.clients, client)
	p.clientMux.Unlock()
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

func (p *connPool) createCodecClient(context context.Context, connData types.CreateConnectionData) str.CodecClient {
	return str.NewCodecClient(context, protocol.HTTP1, connData.Connection, connData.HostInfo)
}

func (p *connPool) report() {
	// report
	go func() {
		for {
			p.clientMux.Lock()
			fmt.Printf("pool = %s, available clients=%d, total clients=%d\n", p.host.Address(), len(p.clients), p.clientCount)
			p.clientMux.Unlock()
			time.Sleep(time.Second)
		}
	}()
}

// stream.CodecClientCallbacks
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	index int //only for troubleshooting

	pool               *connPool
	codecClient        str.CodecClient
	host               types.CreateConnectionData
	totalStream        uint64
	closeWithActiveReq bool
}

func newActiveClient(ctx context.Context, pool *connPool) *activeClient {
	ac := &activeClient{
		index: pool.clientCount,
		pool:  pool,
	}

	data := pool.host.CreateConnection(ctx)
	codecClient := pool.createCodecClient(ctx, data)
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

func (ac *activeClient) OnGoAway() {}
