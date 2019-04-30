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
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/rcrowley/go-metrics"
)

func init() {
	network.RegisterNewPoolFactory(protocol.Xprotocol, NewConnPool)
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

func (p *connPool) CheckAndInit(ctx context.Context) bool {
	return true
}

// DrainConnections no use
func (p *connPool) DrainConnections() {}

// NewStream invoked by Proxy
func (p *connPool) NewStream(context context.Context, responseDecoder types.StreamReceiveListener,
	listener types.PoolEventListener) {
	log.DefaultLogger.Tracef("xprotocol conn pool new stream")

	activeClient := func() *activeClient {
		p.mux.Lock()
		defer p.mux.Unlock()
		if p.primaryClient == nil {
			p.primaryClient = newActiveClient(context, p)
		}
		return p.primaryClient
	}()

	if activeClient == nil {
		listener.OnFailure(types.ConnectionFailure, p.host)
		return
	}

	if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		listener.OnFailure(types.Overflow, p.host)
		p.host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
	} else {
		atomic.AddUint64(&activeClient.totalStream, 1)
		p.host.HostStats().UpstreamRequestTotal.Inc(1)
		p.host.HostStats().UpstreamRequestActive.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
		p.host.ClusterInfo().ResourceManager().Requests().Increase()
		log.DefaultLogger.Tracef("xprotocol conn pool codec client new stream")
		streamSender := activeClient.client.NewStream(context, responseDecoder)
		streamSender.GetStream().AddEventListener(activeClient)

		log.DefaultLogger.Tracef("xprotocol conn pool codec client new stream success,invoked OnPoolReady")
		listener.OnReady(streamSender, p.host)
	}

	return
}

// Close close connection pool
func (p *connPool) Close() {
	p.mux.Lock()
	defer p.mux.Unlock()

	if p.primaryClient != nil {
		p.primaryClient.client.Close()
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

		p.mux.Lock()
		defer p.mux.Unlock()

		if p.primaryClient == client {
			p.primaryClient = nil
		}
	} else if event == types.ConnectTimeout {
		p.host.HostStats().UpstreamRequestTimeout.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestTimeout.Inc(1)
		client.client.Close()
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
	p.host.HostStats().UpstreamConnectionCloseNotify.Inc(1)
	p.host.ClusterInfo().Stats().UpstreamConnectionCloseNotify.Inc(1)

	p.mux.Lock()
	defer p.mux.Unlock()

	if p.primaryClient == client {
		p.movePrimaryToDraining()
	}
}

func (p *connPool) createStreamClient(context context.Context, connData types.CreateConnectionData) str.Client {
	return str.NewStreamClient(context, protocol.Xprotocol, connData.Connection, connData.HostInfo)
}

func (p *connPool) movePrimaryToDraining() {
	if p.drainingClient != nil {
		p.drainingClient.client.Close()
	}

	if p.primaryClient.client.ActiveRequestsNum() == 0 {
		p.primaryClient.client.Close()
	} else {
		p.drainingClient = p.primaryClient
		p.primaryClient = nil
	}
}

// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	pool               *connPool
	client             str.Client
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
	codecClient := pool.createStreamClient(context, data)
	log.DefaultLogger.Tracef("xprotocol new active client , create codec client success")
	codecClient.AddConnectionEventListener(ac)
	codecClient.SetStreamConnectionEventListener(ac)

	ac.client = codecClient
	ac.host = data.HostInfo

	pool.host.HostStats().UpstreamConnectionTotal.Inc(1)
	pool.host.HostStats().UpstreamConnectionActive.Inc(1)
	pool.host.ClusterInfo().Stats().UpstreamConnectionTotal.Inc(1)
	pool.host.ClusterInfo().Stats().UpstreamConnectionActive.Inc(1)

	// bytes total adds all connections data together, but buffered data not
	codecClient.SetConnectionStats(&types.ConnectionStats{
		ReadTotal:     pool.host.ClusterInfo().Stats().UpstreamBytesReadTotal,
		ReadBuffered:  metrics.NewGauge(),
		WriteTotal:    pool.host.ClusterInfo().Stats().UpstreamBytesWriteTotal,
		WriteBuffered: metrics.NewGauge(),
	})

	return ac
}

// types.ConnectionEventListener
// OnEvent handle connection event
func (ac *activeClient) OnEvent(event types.ConnectionEvent) {
	ac.pool.onConnectionEvent(ac, event)
}

// types.StreamEventListener
func (ac *activeClient) OnDestroyStream() {
	ac.pool.onStreamDestroy(ac)
}

func (ac *activeClient) OnResetStream(reason types.StreamResetReason) {
	ac.pool.onStreamReset(ac, reason)
}

// types.StreamConnectionEventListener
// OnGoAway handle go away event
func (ac *activeClient) OnGoAway() {
	ac.pool.onGoAway(ac)
}
