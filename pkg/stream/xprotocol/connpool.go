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

	"mosn.io/mosn/pkg/protocol/xprotocol"

	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	str "mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

const (
	Init = iota
	Connecting
	Connected
)

func init() {
	network.RegisterNewPoolFactory(protocol.Xprotocol, NewConnPool)
	types.RegisterConnPoolFactory(protocol.Xprotocol, true)
}

// types.ConnectionPool
// activeClient used as connected client
// host is the upstream
type connPool struct {
	activeClients sync.Map //sub protocol -> activeClient
	host          atomic.Value
	mux           sync.Mutex
	tlsHash       *types.HashValue
}

// NewConnPool
func NewConnPool(host types.Host) types.ConnectionPool {
	p := &connPool{
		tlsHash: host.TLSHashValue(),
	}
	p.host.Store(host)
	return p
}

func (p *connPool) TLSHashValue() *types.HashValue {
	return p.tlsHash
}

func (p *connPool) init(client *activeClient, sub types.ProtocolName) {
	utils.GoWithRecover(func() {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[stream] [sofarpc] [connpool] init host %s", p.Host().AddressString())
		}

		p.mux.Lock()
		defer p.mux.Unlock()
		client := newActiveClient(context.Background(), sub, p)
		if client != nil {
			client.state = Connected
			p.activeClients.Store(sub, client)
		} else {
			p.activeClients.Delete(sub)
		}
	}, nil)
}

func (p *connPool) Host() types.Host {
	h := p.host.Load()
	if host, ok := h.(types.Host); ok {
		return host
	}

	return nil
}

func (p *connPool) UpdateHost(h types.Host) {
	// TODO: update tls support flag
	p.host.Store(h)
}

func (p *connPool) CheckAndInit(ctx context.Context) bool {
	var client *activeClient

	subProtocol := getSubProtocol(ctx)

	v, ok := p.activeClients.Load(subProtocol)
	if !ok {
		fakeclient := &activeClient{}
		fakeclient.state = Init
		v, _ := p.activeClients.LoadOrStore(subProtocol, fakeclient)
		client = v.(*activeClient)
	} else {
		client = v.(*activeClient)
	}

	if atomic.LoadUint32(&client.state) == Connected {
		return true
	}

	if atomic.CompareAndSwapUint32(&client.state, Init, Connecting) {
		p.init(client, subProtocol)
	}

	return false
}

func (p *connPool) Protocol() types.ProtocolName {
	return protocol.Xprotocol
}

func (p *connPool) NewStream(ctx context.Context,
	responseDecoder types.StreamReceiveListener, listener types.PoolEventListener) {
	subProtocol := getSubProtocol(ctx)

	client, _ := p.activeClients.Load(subProtocol)
	host := p.Host()

	if client == nil {
		listener.OnFailure(types.ConnectionFailure, host)
		return
	}

	activeClient := client.(*activeClient)
	if atomic.LoadUint32(&activeClient.state) != Connected {
		listener.OnFailure(types.ConnectionFailure, host)
		return
	}

	if !host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		listener.OnFailure(types.Overflow, host)
		host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
	} else {
		atomic.AddUint64(&activeClient.totalStream, 1)
		host.HostStats().UpstreamRequestTotal.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)

		var streamEncoder types.StreamSender
		// oneway
		if responseDecoder == nil {
			streamEncoder = activeClient.client.NewStream(ctx, nil)
		} else {
			streamEncoder = activeClient.client.NewStream(ctx, responseDecoder)
			streamEncoder.GetStream().AddEventListener(activeClient)

			host.HostStats().UpstreamRequestActive.Inc(1)
			host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
			host.ClusterInfo().ResourceManager().Requests().Increase()
		}

		listener.OnReady(streamEncoder, host)
	}

	return
}

func (p *connPool) Close() {
	f := func(k, v interface{}) bool {
		ac, _ := v.(*activeClient)
		if ac.client != nil {
			ac.client.Close()
		}
		return true
	}

	p.activeClients.Range(f)
}

// Shutdown stop the keepalive, so the connection will be idle after requests finished
func (p *connPool) Shutdown() {
	f := func(k, v interface{}) bool {
		ac, _ := v.(*activeClient)
		if ac.keepAlive != nil {
			ac.keepAlive.keepAlive.Stop()
		}
		return true
	}
	p.activeClients.Range(f)
}

func (p *connPool) onConnectionEvent(client *activeClient, event api.ConnectionEvent) {
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

			if client.closeWithActiveReq {
				host.HostStats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
				host.ClusterInfo().Stats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
			}

		case api.RemoteClose:
			host.HostStats().UpstreamConnectionRemoteClose.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionRemoteClose.Inc(1)

			if client.closeWithActiveReq {
				host.HostStats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
				host.ClusterInfo().Stats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)

			}
		default:
			// do nothing
		}
		p.mux.Lock()
		p.activeClients.Delete(client.subProtocol)
		p.mux.Unlock()
	} else if event == api.ConnectTimeout {
		host.HostStats().UpstreamRequestTimeout.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTimeout.Inc(1)
		client.client.Close()
	} else if event == api.ConnectFailed {
		host.HostStats().UpstreamConnectionConFail.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionConFail.Inc(1)
	}
}

func (p *connPool) onStreamDestroy(client *activeClient) {
	host := p.Host()
	host.HostStats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().ResourceManager().Requests().Decrease()
}

func (p *connPool) onStreamReset(client *activeClient, reason types.StreamResetReason) {
	host := p.Host()
	if reason == types.StreamConnectionTermination || reason == types.StreamConnectionFailed {
		host.HostStats().UpstreamRequestFailureEject.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestFailureEject.Inc(1)
		client.closeWithActiveReq = true
	} else if reason == types.StreamLocalReset {
		host.HostStats().UpstreamRequestLocalReset.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestLocalReset.Inc(1)
	} else if reason == types.StreamRemoteReset {
		host.HostStats().UpstreamRequestRemoteReset.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestRemoteReset.Inc(1)
	}
}

func (p *connPool) createStreamClient(context context.Context, connData types.CreateConnectionData) str.Client {
	return str.NewStreamClient(context, protocol.Xprotocol, connData.Connection, connData.Host)
}

// keepAliveListener is a types.ConnectionEventListener
type keepAliveListener struct {
	keepAlive types.KeepAlive
}

func (l *keepAliveListener) OnEvent(event api.ConnectionEvent) {
	if event == api.OnReadTimeout {
		l.keepAlive.SendKeepAlive()
	}
}

// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	subProtocol        types.ProtocolName
	pool               *connPool
	keepAlive          *keepAliveListener
	client             str.Client
	host               types.CreateConnectionData
	closeWithActiveReq bool
	totalStream        uint64
	state              uint32
}

func newActiveClient(ctx context.Context, subProtocol types.ProtocolName, pool *connPool) *activeClient {
	ac := &activeClient{
		subProtocol: subProtocol,
		pool:        pool,
	}

	host := pool.Host()
	data := host.CreateConnection(ctx)
	connCtx := mosnctx.WithValue(ctx, types.ContextKeyConnectionID, data.Connection.ID())
	connCtx = mosnctx.WithValue(ctx, types.ContextSubProtocol, string(subProtocol))
	codecClient := pool.createStreamClient(connCtx, data)
	codecClient.AddConnectionEventListener(ac)
	codecClient.SetStreamConnectionEventListener(ac)

	ac.client = codecClient
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
			ac.client.AddConnectionEventListener(ac.keepAlive)
		}
	}

	if err := ac.client.Connect(); err != nil {
		return nil
	}

	// stats
	host.HostStats().UpstreamConnectionTotal.Inc(1)
	host.HostStats().UpstreamConnectionActive.Inc(1)
	host.ClusterInfo().Stats().UpstreamConnectionTotal.Inc(1)
	host.ClusterInfo().Stats().UpstreamConnectionActive.Inc(1)

	// bytes total adds all connections data together
	codecClient.SetConnectionCollector(host.ClusterInfo().Stats().UpstreamBytesReadTotal, host.ClusterInfo().Stats().UpstreamBytesWriteTotal)

	return ac
}

func (ac *activeClient) OnEvent(event api.ConnectionEvent) {
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
func (ac *activeClient) OnGoAway() {}

func getSubProtocol(ctx context.Context) types.ProtocolName {
	if ctx != nil {
		if val := mosnctx.Get(ctx, types.ContextSubProtocol); val != nil {
			if code, ok := val.(string); ok {
				return types.ProtocolName(code)
			}
		}
	}
	return ""
}
