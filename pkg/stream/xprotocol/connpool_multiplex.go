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
	"time"

	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type poolMultiplex struct {
	*connpool

	clientMux     sync.Mutex
	activeClients sync.Map
}

// NewPoolMultiplex generates a multiplex conn pool
func NewPoolMultiplex(p *connpool) types.ConnectionPool {
	return &poolMultiplex{
		connpool:      p,
		activeClients: sync.Map{},
	}
}

func (p *poolMultiplex) init(client *activeClientMultiplex, sub types.ProtocolName) {
	utils.GoWithRecover(func() {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[stream] [sofarpc] [connpool] init host %s", p.Host().AddressString())
		}

		p.clientMux.Lock()
		defer p.clientMux.Unlock()
		client, _ := p.newActiveClient(context.Background(), sub)
		if client != nil {
			client.state = Connected
			p.activeClients.Store(sub, client)
		} else {
			p.activeClients.Delete(sub)
		}
	}, nil)
}

// CheckAndInit init the connection pool
func (p *poolMultiplex) CheckAndInit(ctx context.Context) bool {
	var client *activeClientMultiplex

	subProtocol := getSubProtocol(ctx)

	v, ok := p.activeClients.Load(subProtocol)
	if !ok {
		fakeclient := &activeClientMultiplex{}
		fakeclient.state = Init
		v, _ := p.activeClients.LoadOrStore(subProtocol, fakeclient)
		client = v.(*activeClientMultiplex)
	} else {
		client = v.(*activeClientMultiplex)
	}

	if atomic.LoadUint32(&client.state) == Connected {
		return true
	}

	if atomic.CompareAndSwapUint32(&client.state, Init, Connecting) {
		p.init(client, subProtocol)
	}

	return false
}

// NewStream Create a client stream and call's by proxy
func (p *poolMultiplex) NewStream(ctx context.Context, receiver types.StreamReceiveListener) (types.Host, types.StreamSender, types.PoolFailureReason) {
	subProtocol := getSubProtocol(ctx)

	client, _ := p.activeClients.Load(subProtocol)
	host := p.Host()

	if client == nil {
		return host, nil, types.ConnectionFailure
	}

	activeClient := client.(*activeClientMultiplex)
	if atomic.LoadUint32(&activeClient.state) != Connected {
		return host, nil, types.ConnectionFailure
	}

	if !host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
		return host, nil, types.Overflow
	}

	atomic.AddUint64(&activeClient.totalStream, 1)
	host.HostStats().UpstreamRequestTotal.Inc(1)
	host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)

	var streamEncoder types.StreamSender
	// oneway
	if receiver == nil {
		streamEncoder = activeClient.codecClient.NewStream(ctx, nil)
	} else {
		streamEncoder = activeClient.codecClient.NewStream(ctx, receiver)
		streamEncoder.GetStream().AddEventListener(activeClient)
		host.HostStats().UpstreamRequestActive.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
		host.ClusterInfo().ResourceManager().Requests().Increase()
	}

	return host, streamEncoder, ""
}

// Shutdown stop the keepalive, so the connection will be idle after requests finished
func (p *poolMultiplex) Shutdown() {
	f := func(k, v interface{}) bool {
		ac, _ := v.(*activeClientMultiplex)
		if ac.keepAlive != nil {
			ac.keepAlive.keepAlive.Stop()
		}
		return true
	}
	p.activeClients.Range(f)
}

func (p *poolMultiplex) createStreamClient(context context.Context, connData types.CreateConnectionData) stream.Client {
	return stream.NewStreamClient(context, protocol.Xprotocol, connData.Connection, connData.Host)
}

func (p *poolMultiplex) newActiveClient(ctx context.Context, subProtocol api.Protocol) (*activeClientMultiplex, types.PoolFailureReason) {
	ac := &activeClientMultiplex{
		subProtocol: subProtocol,
		pool:        p,
	}

	host := p.Host()
	data := host.CreateConnection(ctx)
	connCtx := mosnctx.WithValue(ctx, types.ContextKeyConnectionID, data.Connection.ID())
	connCtx = mosnctx.WithValue(connCtx, types.ContextSubProtocol, string(subProtocol))
	codecClient := p.createStreamClient(connCtx, data)
	codecClient.AddConnectionEventListener(ac)
	codecClient.SetStreamConnectionEventListener(ac)

	ac.codecClient = codecClient
	ac.host = data

	// Add Keep Alive
	// protocol is from onNewDetectStream
	if subProtocol != "" {
		// check heartbeat enable, hack: judge trigger result of Heartbeater
		proto := xprotocol.GetProtocol(subProtocol)
		if heartbeater, ok := proto.(xprotocol.Heartbeater); ok && heartbeater.Trigger(0) != nil {
			// create keepalive
			rpcKeepAlive := NewKeepAlive(codecClient, subProtocol, time.Second, 6)
			rpcKeepAlive.StartIdleTimeout()
			ac.keepAlive = &keepAliveListener{
				keepAlive: rpcKeepAlive,
			}
			ac.codecClient.AddConnectionEventListener(ac.keepAlive)
		}
	}

	if err := ac.codecClient.Connect(); err != nil {
		return nil, types.ConnectionFailure
	}

	// stats
	host.HostStats().UpstreamConnectionTotal.Inc(1)
	host.HostStats().UpstreamConnectionActive.Inc(1)
	host.ClusterInfo().Stats().UpstreamConnectionTotal.Inc(1)
	host.ClusterInfo().Stats().UpstreamConnectionActive.Inc(1)

	// bytes total adds all connections data together
	codecClient.SetConnectionCollector(host.ClusterInfo().Stats().UpstreamBytesReadTotal, host.ClusterInfo().Stats().UpstreamBytesWriteTotal)

	return ac, ""
}

func (p *poolMultiplex) Close() {
	f := func(k, v interface{}) bool {
		ac, _ := v.(*activeClientMultiplex)
		if ac.codecClient != nil {
			ac.codecClient.Close()
		}
		return true
	}

	p.activeClients.Range(f)
}

func (p *poolMultiplex) onConnectionEvent(ac *activeClientMultiplex, event api.ConnectionEvent) {
	host := p.Host()
	// event.ConnectFailure() contains types.ConnectTimeout and types.ConnectTimeout
	if event.IsClose() {
		host.HostStats().UpstreamConnectionClose.Inc(1)
		host.HostStats().UpstreamConnectionActive.Dec(1)

		host.ClusterInfo().Stats().UpstreamConnectionClose.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionActive.Dec(1)

		switch event {
		case api.LocalClose:
			host.HostStats().UpstreamConnectionLocalClose.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionLocalClose.Inc(1)

			if ac.closeWithActiveReq {
				host.HostStats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
				host.ClusterInfo().Stats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
			}

		case api.RemoteClose:
			host.HostStats().UpstreamConnectionRemoteClose.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionRemoteClose.Inc(1)

			if ac.closeWithActiveReq {
				host.HostStats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
				host.ClusterInfo().Stats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)

			}
		default:
			// do nothing
		}
		p.clientMux.Lock()
		p.activeClients.Delete(ac.subProtocol)
		p.clientMux.Unlock()
	} else if event == api.ConnectTimeout {
		host.HostStats().UpstreamRequestTimeout.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTimeout.Inc(1)
		ac.codecClient.Close()
	} else if event == api.ConnectFailed {
		host.HostStats().UpstreamConnectionConFail.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionConFail.Inc(1)
	}
}

// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
// nolint: maligned
type activeClientMultiplex struct {
	closeWithActiveReq bool
	totalStream        uint64
	subProtocol        types.ProtocolName
	keepAlive          *keepAliveListener
	state              uint32 // for async connection
	pool               *poolMultiplex
	codecClient        stream.Client
	host               types.CreateConnectionData
}

// types.ConnectionEventListener
// nolint: dupl
func (ac *activeClientMultiplex) OnEvent(event api.ConnectionEvent) {
	ac.pool.onConnectionEvent(ac, event)
}

// types.StreamEventListener
func (ac *activeClientMultiplex) OnDestroyStream() {
	host := ac.pool.Host()
	host.HostStats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().ResourceManager().Requests().Decrease()
}

func (ac *activeClientMultiplex) OnResetStream(reason types.StreamResetReason) {
	host := ac.pool.Host()
	switch reason {
	case types.StreamConnectionTermination, types.StreamConnectionFailed:
		host.HostStats().UpstreamRequestFailureEject.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestFailureEject.Inc(1)
		ac.closeWithActiveReq = true
	case types.StreamLocalReset:
		host.HostStats().UpstreamRequestLocalReset.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestLocalReset.Inc(1)
	case types.StreamRemoteReset:
		host.HostStats().UpstreamRequestRemoteReset.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestRemoteReset.Inc(1)
	}
}

// types.StreamConnectionEventListener
func (ac *activeClientMultiplex) OnGoAway() {}
