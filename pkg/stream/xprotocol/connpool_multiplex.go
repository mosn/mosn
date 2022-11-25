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
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
	"mosn.io/pkg/variable"
)

// poolMultiplex is used for multiplex protocols like sofa, dubbo, etc.
// a single pool is connections which can be reused in a single host
type poolMultiplex struct {
	*connpool

	clientMux              sync.Mutex
	activeClients          []sync.Map // TODO: do not need map anymore
	currentCheckAndInitIdx int64

	shutdown bool // pool is already shutdown
}

var (
	defaultMaxConn         = 1
	connNumberLimit uint64 = 65535 // port limit
)

func isValidMaxNum(maxConns uint64) bool {
	// xDS cluster if not limit max connection will recv:
	// max_connections:{value:4294967295}  max_pending_requests:{value:4294967295}  max_requests:{value:4294967295}  max_retries:{value:4294967295}
	// if not judge max, will oom
	return maxConns > 0 && maxConns < connNumberLimit
}

// SetDefaultMaxConnNumPerHostPortForMuxPool set the max connections for each host:port
// users could use this function or cluster threshold config to configure connection no.
func SetDefaultMaxConnNumPerHostPortForMuxPool(maxConns int) {
	if isValidMaxNum(uint64(maxConns)) {
		defaultMaxConn = maxConns
	}
}

// NewPoolMultiplex generates a multiplex conn pool
func NewPoolMultiplex(p *connpool) types.ConnectionPool {
	maxConns := p.Host().ClusterInfo().ResourceManager().Connections().Max()

	// the cluster threshold config has higher privilege than global config
	// if a valid number is provided, should use it
	if !isValidMaxNum(maxConns) {
		// override maxConns by default max conns
		maxConns = uint64(defaultMaxConn)
	}

	return &poolMultiplex{
		connpool:      p,
		activeClients: make([]sync.Map, maxConns),
	}
}

func (p *poolMultiplex) init(sub types.ProtocolName, index int) {
	utils.GoWithRecover(func() {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[stream] [sofarpc] [connpool] init host %s", p.Host().AddressString())
		}

		p.clientMux.Lock()
		defer p.clientMux.Unlock()

		// if the pool is already shut down, do nothing directly return
		if p.shutdown {
			return
		}
		ctx := context.Background() // TODO: a new context ?
		client, _ := p.newActiveClient(ctx, sub)
		if client != nil {
			client.state = Connected
			client.indexInPool = index
			p.activeClients[index].Store(sub, client)
		} else {
			p.activeClients[index].Delete(sub)
		}
	}, nil)
}

// CheckAndInit init the connection pool
func (p *poolMultiplex) CheckAndInit(ctx context.Context) bool {
	var clientIdx int64 = 0 // most use cases, there will only be 1 connection
	if len(p.activeClients) > 1 {
		if clientIdx = getClientIDFromDownStreamCtx(ctx); clientIdx == invalidClientID {
			clientIdx = atomic.AddInt64(&p.currentCheckAndInitIdx, 1) % int64(len(p.activeClients))
			// set current client index to downstream context
			_ = variable.Set(ctx, types.VariableConnectionPoolIndex, clientIdx)
		}
	}

	var client *activeClientMultiplex
	subProtocol := p.connpool.codec.ProtocolName()

	v, ok := p.activeClients[clientIdx].Load(subProtocol)
	if !ok {
		fakeclient := &activeClientMultiplex{}
		fakeclient.state = Init
		v, _ := p.activeClients[clientIdx].LoadOrStore(subProtocol, fakeclient)
		client = v.(*activeClientMultiplex)
	} else {
		client = v.(*activeClientMultiplex)
	}

	if atomic.LoadUint32(&client.state) == Connected {
		return true
	}

	// init connection when client is Init or GoAway.
	if atomic.CompareAndSwapUint32(&client.state, Init, Connecting) ||
		atomic.CompareAndSwapUint32(&client.state, GoAway, Connecting) {
		p.init(subProtocol, int(clientIdx))
	}

	return false
}

// NewStream Create a client stream and call's by proxy
func (p *poolMultiplex) NewStream(ctx context.Context, receiver types.StreamReceiveListener) (types.Host, types.StreamSender, types.PoolFailureReason) {
	var (
		ok        bool
		clientIdx int64 = 0
	)

	if len(p.activeClients) > 1 {
		clientIdxInter, _ := variable.Get(ctx, types.VariableConnectionPoolIndex)
		if clientIdx, ok = clientIdxInter.(int64); !ok {
			// this client is not inited
			return p.Host(), nil, types.ConnectionFailure
		}
	}

	subProtocol := p.connpool.codec.ProtocolName()

	client, _ := p.activeClients[clientIdx].Load(subProtocol)

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

	_ = variable.Set(ctx, types.VariableUpstreamConnectionID, activeClient.codecClient.ConnID())

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
	utils.GoWithRecover(func() {
		{
			p.clientMux.Lock()
			if p.shutdown {
				p.clientMux.Unlock()
				return
			}
			p.shutdown = true
			p.clientMux.Unlock()
		}

		for i := 0; i < len(p.activeClients); i++ {
			f := func(k, v interface{}) bool {
				ac, _ := v.(*activeClientMultiplex)
				if ac.keepAlive != nil {
					ac.keepAlive.keepAlive.Stop()
				}
				return true
			}
			p.activeClients[i].Range(f)
		}

	}, nil)
}

func (p *poolMultiplex) createStreamClient(context context.Context, connData types.CreateConnectionData) stream.Client {
	return stream.NewStreamClient(context, p.connpool.protocol, connData.Connection, connData.Host)
}

func (p *poolMultiplex) newActiveClient(ctx context.Context, subProtocol api.ProtocolName) (*activeClientMultiplex, types.PoolFailureReason) {
	ac := &activeClientMultiplex{
		subProtocol: subProtocol,
		pool:        p,
	}

	host := p.Host()
	data := host.CreateConnection(ctx)
	connCtx := ctx
	_ = variable.Set(ctx, types.VariableConnectionID, data.Connection.ID())
	codecClient := p.createStreamClient(connCtx, data)
	codecClient.AddConnectionEventListener(ac)
	codecClient.SetStreamConnectionEventListener(ac)

	ac.codecClient = codecClient
	ac.host = data

	// Add Keep Alive
	// protocol is from onNewDetectStream
	if subProtocol != "" {
		// check heartbeat enable, hack: judge trigger result of Heartbeater
		proto := p.connpool.codec.NewXProtocol(ctx)
		if heartbeater, ok := proto.(api.Heartbeater); ok && heartbeater.Trigger(ctx, 0) != nil {
			// create keepalive
			rpcKeepAlive := NewKeepAlive(codecClient, proto, time.Second)
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
	for i := 0; i < len(p.activeClients); i++ {
		f := func(k, v interface{}) bool {
			ac, _ := v.(*activeClientMultiplex)
			if ac.codecClient != nil {
				ac.codecClient.Close()
			}
			return true
		}

		p.activeClients[i].Range(f)
	}
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
		// only delete the active client when the state is not GoAway
		// since the goaway state client has already been overwritten.
		if atomic.LoadUint32(&ac.state) != GoAway {
			p.clientMux.Lock()
			p.activeClients[ac.indexInPool].Delete(ac.subProtocol)
			p.clientMux.Unlock()
		}
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
	indexInPool        int
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

	if atomic.LoadUint32(&ac.state) == GoAway && ac.codecClient.ActiveRequestsNum() == 0 {
		ac.codecClient.Close()
	}
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
func (ac *activeClientMultiplex) OnGoAway() {
	atomic.StoreUint32(&ac.state, GoAway)

	if ac.codecClient.ActiveRequestsNum() == 0 {
		ac.codecClient.Close()
	}
}

const invalidClientID = -1

func getClientIDFromDownStreamCtx(ctx context.Context) int64 {
	clientIdxInter, _ := variable.Get(ctx, types.VariableConnectionPoolIndex)
	clientIdx, ok := clientIdxInter.(int64)
	if !ok {
		return invalidClientID
	}
	return clientIdx
}
