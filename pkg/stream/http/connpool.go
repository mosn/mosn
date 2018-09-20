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
	"github.com/alipay/sofa-mosn/pkg/stats"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	metrics "github.com/rcrowley/go-metrics"
)

func init() {
	proxy.RegisterNewPoolFactory(protocol.HTTP1, NewConnPool)
	types.RegisterConnPoolFactory(protocol.HTTP1, true)

}

// types.ConnectionPool
type connPool struct {
	host       types.Host
	client     *activeClient
	initClient sync.Once
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
func (p *connPool) NewStream(context context.Context, streamID string, responseDecoder types.StreamReceiver,
	cb types.PoolEventListener) types.Cancellable {

	if p.client == nil {
		p.initClient.Do(func() {
			p.client = newActiveClient(context, p)
		})
	}

	if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		cb.OnFailure(streamID, types.Overflow, nil)
		p.host.HostStats().Counter(stats.UpstreamRequestPendingOverflow).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestPendingOverflow).Inc(1)
	} else {
		p.host.HostStats().Counter(stats.UpstreamRequestTotal).Inc(1)
		p.host.HostStats().Counter(stats.UpstreamRequestActive).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestTotal).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestActive).Inc(1)
		p.host.ClusterInfo().ResourceManager().Requests().Increase()

		streamEncoder := p.client.codecClient.NewStream(context, streamID, responseDecoder)
		cb.OnReady(streamID, streamEncoder, p.host)
	}

	return nil
}

func (p *connPool) Close() {
	p.client = nil
}

func (p *connPool) onConnectionEvent(client *activeClient, event types.ConnectionEvent) {
	if event.IsClose() {

		if client.closeWithActiveReq {
			if event == types.LocalClose {
				p.host.HostStats().Counter(stats.UpstreamConnectionLocalCloseWithActiveRequest).Inc(1)
				p.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionLocalCloseWithActiveRequest).Inc(1)
			} else if event == types.RemoteClose {
				p.host.HostStats().Counter(stats.UpstreamConnectionRemoteCloseWithActiveRequest).Inc(1)
				p.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionRemoteCloseWithActiveRequest).Inc(1)
			}
		}

		if p.client == client {
			p.client = nil
		}
	} else if event == types.ConnectTimeout {
		p.host.HostStats().Counter(stats.UpstreamRequestTimeout).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestTimeout).Inc(1)
		client.codecClient.Close()
	} else if event == types.ConnectFailed {
		p.host.HostStats().Counter(stats.UpstreamConnectionConFail).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionConFail).Inc(1)
	}
}

func (p *connPool) onStreamDestroy(client *activeClient) {
	p.host.HostStats().Counter(stats.UpstreamRequestActive).Dec(1)
	p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestActive).Dec(1)
	p.host.ClusterInfo().ResourceManager().Requests().Decrease()
}

func (p *connPool) onStreamReset(client *activeClient, reason types.StreamResetReason) {
	if reason == types.StreamConnectionTermination || reason == types.StreamConnectionFailed {
		p.host.HostStats().Counter(stats.UpstreamRequestFailureEject).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestFailureEject).Inc(1)
		client.closeWithActiveReq = true
	} else if reason == types.StreamLocalReset {
		p.host.HostStats().Counter(stats.UpstreamRequestLocalReset).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestLocalReset).Inc(1)
	} else if reason == types.StreamRemoteReset {
		p.host.HostStats().Counter(stats.UpstreamRequestRemoteReset).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestRemoteReset).Inc(1)
	}
}

func (p *connPool) onGoAway(client *activeClient) {
	p.host.HostStats().Counter(stats.UpstreamConnectionCloseNotify).Inc(1)
	p.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionCloseNotify).Inc(1)

	// http/1.x should not enter this branch
	//
	//if p.primaryClient == client {
	//	p.movePrimaryToDraining()
	//}
}

// stream.CodecClientCallbacks
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	pool               *connPool
	codecClient        str.CodecClient
	host               types.HostInfo
	totalStream        uint64
	closeWithActiveReq bool
}

func newActiveClient(context context.Context, pool *connPool) *activeClient {
	ac := &activeClient{
		pool: pool,
	}
	//
	//data := pool.host.CreateConnection(context)
	//data.Connection.Connect(false)

	codecClient := NewHTTP1CodecClient(context, pool.host)
	codecClient.AddConnectionCallbacks(ac)
	codecClient.SetCodecClientCallbacks(ac)
	codecClient.SetCodecConnectionCallbacks(ac)

	ac.codecClient = codecClient
	ac.host = pool.host

	pool.host.HostStats().Counter(stats.UpstreamConnectionTotal).Inc(1)
	pool.host.HostStats().Counter(stats.UpstreamConnectionActive).Inc(1)
	//pool.host.HostStats().Counter(UpstreamConnectionTotalHTTP1).Inc(1)
	pool.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionTotal).Inc(1)
	pool.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionActive).Inc(1)
	//pool.host.ClusterInfo().Stats().Counter(UpstreamConnectionTotalHTTP1).Inc(1)

	// bytes total adds all connections data together, but buffered data not
	codecClient.SetConnectionStats(&types.ConnectionStats{
		ReadTotal:     pool.host.ClusterInfo().Stats().Counter(stats.UpstreamBytesReadTotal),
		ReadBuffered:  metrics.NewGauge(),
		WriteTotal:    pool.host.ClusterInfo().Stats().Counter(stats.UpstreamBytesWriteTotal),
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
