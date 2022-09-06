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

	atomicex "go.uber.org/atomic"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

// poolPingPong is used for ping pong protocol such as http
// which must keep reading connection, and wait for the upstream to response before sending the next request
type poolPingPong struct {
	*connpool

	totalClientCount atomicex.Uint64 // total clients
	clientMux        sync.Mutex
	idleClients      []*activeClientPingPong
}

// NewPoolPingPong generates a connection pool which uses p pingpong protocol
func NewPoolPingPong(p *connpool) types.ConnectionPool {
	return &poolPingPong{
		connpool:    p,
		idleClients: []*activeClientPingPong{},
	}
}

// CheckAndInit init the connection pool
func (p *poolPingPong) CheckAndInit(ctx context.Context) bool {
	return true
}

// NewStream Create a client stream and call's by proxy
func (p *poolPingPong) NewStream(ctx context.Context, receiver types.StreamReceiveListener) (types.Host, types.StreamSender, types.PoolFailureReason) {
	host := p.Host()

	c, reason := p.GetActiveClient(ctx)
	if reason != "" {
		return host, nil, reason
	}
	_ = variable.Set(ctx, types.VariableUpstreamConnectionID, c.codecClient.ConnID())

	var streamSender = c.codecClient.NewStream(ctx, receiver)

	streamSender.GetStream().AddEventListener(c) // OnResetStream, OnDestroyStream

	// FIXME one way
	// is there any need to skip the metrics?
	if receiver == nil {
		return host, streamSender, ""
	}

	host.HostStats().UpstreamRequestActive.Inc(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
	host.ClusterInfo().ResourceManager().Requests().Increase()

	return host, streamSender, ""
}

// GetActiveClient get a avail client
// nolint: dupl
func (p *poolPingPong) GetActiveClient(ctx context.Context) (*activeClientPingPong, types.PoolFailureReason) {

	host := p.Host()
	if !host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
		return nil, types.Overflow
	}

	p.clientMux.Lock()

	proto := p.connpool.codec.ProtocolName()

	n := len(p.idleClients)

	// max conns is 0 means no limit
	maxConns := host.ClusterInfo().ResourceManager().Connections().Max()
	// no available client
	var (
		c      *activeClientPingPong
		reason types.PoolFailureReason
	)

	if n == 0 { // nolint: nestif
		if maxConns == 0 || p.totalClientCount.Load() < maxConns {
			// connection not multiplex,
			// so we can concurrently build connections here
			p.clientMux.Unlock()
			c, reason = p.newActiveClient(ctx, proto)
			if c != nil && reason == "" {
				p.totalClientCount.Inc()
			}

			goto RET
		} else {
			p.clientMux.Unlock()

			host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
			host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
			c, reason = nil, types.Overflow

			goto RET
		}
	} else {
		defer p.clientMux.Unlock()

		var lastIdx = n - 1
		// Only refuse extra connection, keepalive-connection is closed by timeout
		usedConns := p.totalClientCount.Load() - uint64(n) + 1
		if maxConns != 0 && usedConns > host.ClusterInfo().ResourceManager().Connections().Max() {
			host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
			host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
			c, reason = nil, types.Overflow
			goto RET
		}

		c = p.idleClients[lastIdx]
		p.idleClients[lastIdx] = nil
		p.idleClients = p.idleClients[:lastIdx]

		goto RET
	}

RET:

	if c != nil && reason == "" {
		host.HostStats().UpstreamRequestTotal.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)
	}

	return c, reason
}

func (p *poolPingPong) Close() {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	for _, c := range p.idleClients {
		c.host.Connection.Close(api.NoFlush, api.LocalClose)
	}
}

func (p *poolPingPong) Shutdown() {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	for _, c := range p.idleClients {
		c.OnGoAway()
		if c.keepAlive != nil {
			c.keepAlive.keepAlive.Stop()
		}
	}
}

// return client to pool
func (p *poolPingPong) putClientToPoolLocked(client *activeClientPingPong) {

	if !client.closed {
		p.idleClients = append(p.idleClients, client)
	}
}

func (p *poolPingPong) newActiveClient(ctx context.Context, subProtocol api.ProtocolName) (*activeClientPingPong, types.PoolFailureReason) {
	ac := &activeClientPingPong{
		pool:        p,
		subProtocol: subProtocol,
		host:        p.Host().CreateConnection(ctx),
	}

	host := p.Host()
	connCtx := ctx

	ac.host.Connection.AddConnectionEventListener(ac)

	// first connect to dest addr, then create stream client
	if err := ac.host.Connection.Connect(); err != nil {
		return nil, types.ConnectionFailure
	}

	////////// codec client
	codecClient := stream.NewStreamClient(connCtx, p.protocol, ac.host.Connection, host)
	codecClient.SetStreamConnectionEventListener(ac) // ac.OnGoAway
	ac.codecClient = codecClient
	// bytes total adds all connections data together
	codecClient.SetConnectionCollector(host.ClusterInfo().Stats().UpstreamBytesReadTotal, host.ClusterInfo().Stats().UpstreamBytesWriteTotal)

	// Add Keep Alive
	// protocol is from onNewDetectStream
	// check heartbeat enable, hack: judge trigger result of Heartbeater
	proto := p.connpool.codec.NewXProtocol(ctx)
	if heartbeater, ok := proto.(api.Heartbeater); ok && heartbeater.Trigger(ctx, 0) != nil {
		// create keepalive
		rpcKeepAlive := NewKeepAlive(ac.codecClient, proto, time.Second)
		rpcKeepAlive.StartIdleTimeout()

		ac.SetHeartBeater(rpcKeepAlive)
	}
	////////// codec client

	atomic.StoreUint32(&ac.state, Connected)

	// stats
	host.HostStats().UpstreamConnectionTotal.Inc(1)
	host.HostStats().UpstreamConnectionActive.Inc(1)
	host.ClusterInfo().Stats().UpstreamConnectionTotal.Inc(1)
	host.ClusterInfo().Stats().UpstreamConnectionActive.Inc(1)

	return ac, ""
}

// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
// nolint: maligned
type activeClientPingPong struct {
	closeWithActiveReq bool

	closed          bool
	shouldCloseConn bool
	subProtocol     types.ProtocolName
	keepAlive       *keepAliveListener
	state           uint32 // for async connection

	pool        *poolPingPong
	codecClient stream.Client
	host        types.CreateConnectionData
}

// Close return this client back to pool
func (ac *activeClientPingPong) Close(err error) {
	if err != nil {
		// if pool is not using multiplex mode
		// this conn is not in the pool
		ac.host.Connection.Close(api.NoFlush, api.LocalClose)
		return
	}

	// return to pool
	ac.pool.clientMux.Lock()
	defer ac.pool.clientMux.Unlock()
	ac.pool.putClientToPoolLocked(ac)
}

// removeFromPool removes this client from connection pool
func (ac *activeClientPingPong) removeFromPool() {
	p := ac.pool
	p.clientMux.Lock()

	defer p.clientMux.Unlock()
	p.totalClientCount.Dec()
	for idx, c := range p.idleClients {
		if c == ac {
			// remove this element
			lastIdx := len(p.idleClients) - 1
			// 	1. swap this with the last
			p.idleClients[idx], p.idleClients[lastIdx] =
				p.idleClients[lastIdx], p.idleClients[idx]
			// 	2. set last to nil
			p.idleClients[lastIdx] = nil
			// 	3. remove the last
			p.idleClients = p.idleClients[:lastIdx]
		}
	}
	ac.closed = true
}

// types.ConnectionEventListener
// nolint: dupl
func (ac *activeClientPingPong) OnEvent(event api.ConnectionEvent) {
	p := ac.pool

	host := p.Host()
	// all protocol should report the following metrics
	if ac.closeWithActiveReq {
		if event == api.LocalClose {
			host.HostStats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
		} else if event == api.RemoteClose {
			host.HostStats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
		}
	}

	switch {
	case event.IsClose():
		// if p.protocol == protocol.Xprotocol {
		host.HostStats().UpstreamConnectionClose.Inc(1)
		host.HostStats().UpstreamConnectionActive.Dec(1)
		host.ClusterInfo().Stats().UpstreamConnectionClose.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionActive.Dec(1)

		switch event { // nolint: exhaustive
		case api.LocalClose:
			host.HostStats().UpstreamConnectionLocalClose.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionLocalClose.Inc(1)
		case api.RemoteClose:
			host.HostStats().UpstreamConnectionRemoteClose.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionRemoteClose.Inc(1)
		default:
			// do nothing
		}
		// }

		// RemoteClose when read/write error
		// LocalClose when there is a panic
		// OnReadErrClose when read failed
		ac.removeFromPool()

	case event == api.ConnectTimeout:
		host.HostStats().UpstreamRequestTimeout.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTimeout.Inc(1)
	case event == api.ConnectFailed:
		host.HostStats().UpstreamConnectionConFail.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionConFail.Inc(1)
	default:
		// do nothing
	}
}

// types.StreamEventListener
func (ac *activeClientPingPong) OnDestroyStream() {
	host := ac.pool.Host()
	host.HostStats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().ResourceManager().Requests().Decrease()

	ac.Close(nil)
}

func (ac *activeClientPingPong) OnResetStream(reason types.StreamResetReason) {
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

	if reason == types.StreamLocalReset && !ac.closed {
		// for xprotocol ping pong
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[stream] [pingpong] stream local reset, blow codecClient away also, Connection = %d",
				ac.host.Connection.ID())
		}
		ac.shouldCloseConn = true
	}
}

// types.StreamConnectionEventListener
func (ac *activeClientPingPong) OnGoAway() {
	ac.shouldCloseConn = true
}

// SetHeartBeater set the heart beat for an active client
func (ac *activeClientPingPong) SetHeartBeater(hb types.KeepAlive) {
	// clear the previous keepAlive
	if ac.keepAlive != nil && ac.keepAlive.keepAlive != nil {
		ac.keepAlive.keepAlive.Stop()
		ac.keepAlive = nil
	}

	ac.keepAlive = &keepAliveListener{
		keepAlive: hb,
		conn:      ac.host.Connection,
	}

	ac.host.Connection.AddConnectionEventListener(ac.keepAlive)
}
