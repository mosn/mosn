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

package xprotocol

import (
	"context"
	"sync"

	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/proxy"
	"github.com/alipay/sofa-mosn/pkg/stats"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	proxy.RegisterNewPoolFactory(protocol.Xprotocol, NewConnPool)
	types.RegisterConnPoolFactory(protocol.Xprotocol, true)

}

// types.ConnectionPool
type connPool struct {
	primaryClient  *activeClient
	drainingClient *activeClient
	mux            sync.Mutex
	host           types.Host
}

// NewConnPool for xprotocol upstream host
func NewConnPool(host types.Host) types.ConnectionPool {
	return &connPool{
		host: host,
	}
}

// Protocol return xprotocol
func (p *connPool) Protocol() types.Protocol {
	return protocol.Xprotocol
}

// DrainConnections no use
func (p *connPool) DrainConnections() {}

// NewStream invoked by Proxy
func (p *connPool) NewStream(context context.Context, streamID string, responseDecoder types.StreamReceiver,
	cb types.PoolEventListener) types.Cancellable {
	log.DefaultLogger.Tracef("xprotocol conn pool new stream")
	p.mux.Lock()

	if p.primaryClient == nil {
		p.primaryClient = newActiveClient(context, p)
	}
	p.mux.Unlock()

	if p.primaryClient == nil {
		cb.OnFailure(streamID, types.ConnectionFailure, nil)
		return nil
	}

	if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		cb.OnFailure(streamID, types.Overflow, nil)
		p.host.HostStats().Counter(stats.UpstreamRequestPendingOverflow).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestPendingOverflow).Inc(1)
	} else {
		atomic.AddUint64(&p.primaryClient.totalStream, 1)
		p.host.HostStats().Counter(stats.UpstreamRequestTotal).Inc(1)
		p.host.HostStats().Counter(stats.UpstreamRequestActive).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestTotal).Inc(1)
		p.host.ClusterInfo().Stats().Counter(stats.UpstreamRequestActive).Inc(1)
		p.host.ClusterInfo().ResourceManager().Requests().Increase()
		log.DefaultLogger.Tracef("xprotocol conn pool codec client new stream")
		streamEncoder := p.primaryClient.codecClient.NewStream(context, streamID, responseDecoder)
		log.DefaultLogger.Tracef("xprotocol conn pool codec client new stream success,invoked OnPoolReady")
		cb.OnReady(streamID, streamEncoder, p.host)
	}

	return nil
}

// Close close connection pool
func (p *connPool) Close() {
	p.mux.Lock()
	defer p.mux.Unlock()

	if p.primaryClient != nil {
		p.primaryClient.codecClient.Close()
	}
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

		p.mux.Lock()
		defer p.mux.Unlock()

		if p.primaryClient == client {
			p.primaryClient = nil
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

	p.mux.Lock()
	defer p.mux.Unlock()

	if p.primaryClient == client {
		p.movePrimaryToDraining()
	}
}

func (p *connPool) createCodecClient(context context.Context, connData types.CreateConnectionData) str.CodecClient {
	return str.NewCodecClient(context, protocol.Xprotocol, connData.Connection, connData.HostInfo)
}

func (p *connPool) movePrimaryToDraining() {
	if p.drainingClient != nil {
		p.drainingClient.codecClient.Close()
	}

	if p.primaryClient.codecClient.ActiveRequestsNum() == 0 {
		p.primaryClient.codecClient.Close()
	} else {
		p.drainingClient = p.primaryClient
		p.primaryClient = nil
	}
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

	log.DefaultLogger.Tracef("xprotocol new active client , try to create connection")
	data := pool.host.CreateConnection(context)
	data.Connection.Connect(true)
	log.DefaultLogger.Tracef("xprotocol new active client , connect success %v", data)

	log.DefaultLogger.Tracef("xprotocol new active client , try to create codec client")
	codecClient := pool.createCodecClient(context, data)
	log.DefaultLogger.Tracef("xprotocol new active client , create codec client success")
	codecClient.AddConnectionCallbacks(ac)
	codecClient.SetCodecClientCallbacks(ac)
	codecClient.SetCodecConnectionCallbacks(ac)

	ac.codecClient = codecClient
	ac.host = data.HostInfo

	pool.host.HostStats().Counter(stats.UpstreamConnectionTotal).Inc(1)
	pool.host.HostStats().Counter(stats.UpstreamConnectionActive).Inc(1)
	//pool.host.HostStats().Counter(stats.UpstreamConnectionTotalHTTP2).Inc(1)
	pool.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionTotal).Inc(1)
	pool.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionActive).Inc(1)
	//pool.host.ClusterInfo().Stats().Counter(stats.UpstreamConnectionTotalHTTP2).Inc(1)

	codecClient.SetConnectionStats(&types.ConnectionStats{
		ReadTotal:    pool.host.ClusterInfo().Stats().Counter(stats.UpstreamBytesRead),
		ReadCurrent:  pool.host.ClusterInfo().Stats().Gauge(stats.UpstreamBytesReadCurrent),
		WriteTotal:   pool.host.ClusterInfo().Stats().Counter(stats.UpstreamBytesWrite),
		WriteCurrent: pool.host.ClusterInfo().Stats().Gauge(stats.UpstreamBytesWriteCurrent),
	})

	return ac
}

// OnEvent handle connection event
func (ac *activeClient) OnEvent(event types.ConnectionEvent) {
	ac.pool.onConnectionEvent(ac, event)
}

// OnAboveWriteBufferHighWatermark no use
func (ac *activeClient) OnAboveWriteBufferHighWatermark() {}

// OnBelowWriteBufferLowWatermark no use
func (ac *activeClient) OnBelowWriteBufferLowWatermark() {}

// OnStreamDestroy destroy stream
func (ac *activeClient) OnStreamDestroy() {
	ac.pool.onStreamDestroy(ac)
}

// OnStreamReset reset stream
func (ac *activeClient) OnStreamReset(reason types.StreamResetReason) {
	ac.pool.onStreamReset(ac, reason)
}

// OnGoAway handle go away event
func (ac *activeClient) OnGoAway() {
	ac.pool.onGoAway(ac)
}
