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

package sofarpc

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/proxy"
	"github.com/alipay/sofa-mosn/pkg/stats"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	proxy.RegisterNewPoolFactory(protocol.SofaRPC, NewConnPool)
	types.RegisterConnPoolFactory(protocol.SofaRPC, true)

}

// types.ConnectionPool
// activeClient used as connected client
// host is the upstream
type connPool struct {
	activeClient *activeClient
	host         types.Host

	mux sync.Mutex
}

// NewConnPool
func NewConnPool(host types.Host) types.ConnectionPool {
	return &connPool{
		host: host,
	}
}

func (p *connPool) Protocol() types.Protocol {
	return protocol.SofaRPC
}

func (p *connPool) NewStream(context context.Context, streamID string,
	responseDecoder types.StreamReceiver, cb types.PoolEventListener) types.Cancellable {
	p.mux.Lock()
	if p.activeClient == nil {
		p.activeClient = newActiveClient(context, p)
	}

	p.mux.Unlock()

	activeClient := p.activeClient
	if activeClient == nil {
		cb.OnFailure(streamID, types.ConnectionFailure, nil)
		return nil
	}

	if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		cb.OnFailure(streamID, types.Overflow, nil)
		p.host.HostStats().Counter(stats.UpstreamRequestPendingOverflow).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestPendingOverflow).Inc(1)
	} else {
		atomic.AddUint64(&activeClient.totalStream, 1)
		p.host.HostStats().Counter(stats.UpstreamRequestTotal).Inc(1)
		p.host.HostStats().Counter(stats.UpstreamRequestActive).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestTotal).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestActive).Inc(1)
		p.host.ClusterInfo().ResourceManager().Requests().Increase()
		streamEncoder := activeClient.codecClient.NewStream(context, streamID, responseDecoder)
		cb.OnReady(streamID, streamEncoder, p.host)
	}

	return nil
}

func (p *connPool) Close() {
	if p.activeClient != nil {
		p.activeClient.codecClient.Close()
	}
}

func (p *connPool) onConnectionEvent(client *activeClient, event types.ConnectionEvent) {
	// event.ConnectFailure() contains types.ConnectTimeout and types.ConnectTimeout
	if event.IsClose() {
		if client.closeWithActiveReq {
			if event == types.LocalClose {
				p.host.HostStats().Counter(stats.UpstreamConnectionLocalCloseWithActiveRequest).Inc(1)
				p.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionLocalCloseWithActiveRequest).Inc(1)
			} else if event == types.RemoteClose {
				p.host.HostStats().Counter(stats.UpstreamConnectionRemoteCloseWithActiveRequest).Inc(1)
				p.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionRemoteCloseWithActiveRequest).Inc(1)
			}
			p.activeClient = nil
		}
	} else if event == types.ConnectTimeout {
		p.host.HostStats().Counter(stats.UpstreamRequestTimeout).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestTimeout).Inc(1)
		client.codecClient.Close()
		p.activeClient = nil
	} else if event == types.ConnectFailed {
		p.host.HostStats().Counter(stats.UpstreamConnectionConFail).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionConFail).Inc(1)
		p.activeClient = nil
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

func (p *connPool) createCodecClient(context context.Context, connData types.CreateConnectionData) str.CodecClient {
	return str.NewCodecClient(context, protocol.SofaRPC, connData.Connection, connData.HostInfo)
}

// stream.CodecClientCallbacks
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	pool               *connPool
	codecClient        str.CodecClient
	host               types.CreateConnectionData
	closeWithActiveReq bool
	totalStream        uint64
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
	pool.host.HostStats().Counter(stats.UpstreamConnectionTotal).Inc(1)
	pool.host.HostStats().Counter(stats.UpstreamConnectionActive).Inc(1)
	//pool.host.HostStats().Counter(UpstreamConnectionTotalRPC).Inc(1)
	pool.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionTotal).Inc(1)
	pool.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionActive).Inc(1)
	//pool.host.ClusterInfo().Stats().Counter(UpstreamConnectionTotalRPC).Inc(1)
	codecClient.SetConnectionStats(&types.ConnectionStats{
		ReadTotal:    pool.host.ClusterInfo().Stats().Counter(stats.UpstreamBytesRead),
		ReadCurrent:  pool.host.ClusterInfo().Stats().Gauge(stats.UpstreamBytesReadCurrent),
		WriteTotal:   pool.host.ClusterInfo().Stats().Counter(stats.UpstreamBytesWrite),
		WriteCurrent: pool.host.ClusterInfo().Stats().Gauge(stats.UpstreamBytesWriteCurrent),
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
