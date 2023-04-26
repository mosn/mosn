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
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

// poolBinding is a special purpose connection pool,
// used to bind the downstream conn with the upstream conn, which satisfies following condition
// 1. if upstream connection is closed, the corresponding downstream connection should also be closed, and vice versa
// 2. the downstream data on a specific connection should always send to the same upstream connection
// should not use it until you clearly understand what you are doing
type poolBinding struct {
	*connpool

	clientMux   sync.Mutex
	idleClients map[uint64]*activeClientBinding // downstream connection id --> upstream client
}

// NewPoolBinding generates a binding connection pool
// the upstream connection close will trigger the downstream connection to close and vice versa
func NewPoolBinding(p *connpool) types.ConnectionPool {
	return &poolBinding{
		connpool:    p,
		idleClients: make(map[uint64]*activeClientBinding),
	}
}

// CheckAndInit init the connection pool
func (p *poolBinding) CheckAndInit(ctx context.Context) bool {
	return true
}

type downstreamCloseListener struct {
	upstreamClient *activeClientBinding
}

var downstreamClosed = errors.New("downstream closed conn")

func (d downstreamCloseListener) OnEvent(event api.ConnectionEvent) {
	if event.IsClose() && d.upstreamClient != nil {
		d.upstreamClient.Close(downstreamClosed)
	}
}

// NewStream Create a client stream and call's by proxy
func (p *poolBinding) NewStream(ctx context.Context, receiver types.StreamReceiveListener) (types.Host, types.StreamSender, types.PoolFailureReason) {
	host := p.Host()
	// stats
	host.HostStats().UpstreamRequestTotal.Inc(1)
	host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)

	c, reason := p.GetActiveClient(ctx)

	if reason != "" {
		return host, nil, reason
	}

	c.addDownConnListenerOnce(ctx)

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
func (p *poolBinding) GetActiveClient(ctx context.Context) (*activeClientBinding, types.PoolFailureReason) {
	host := p.Host()
	if !host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
		return nil, types.Overflow
	}

	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	downstreamConnID := getDownstreamConnID(ctx)
	if c, ok := p.idleClients[downstreamConnID]; ok {
		// the client was already initialized
		return c, ""
	}

	// no available client
	c, reason := p.newActiveClient(ctx)
	if c != nil && reason == "" {
		p.idleClients[downstreamConnID] = c
	}

	return c, reason
}

func (p *poolBinding) Close() {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	for _, c := range p.idleClients {
		c.host.Connection.Close(api.NoFlush, api.LocalClose)
	}
}

func (p *poolBinding) Shutdown() {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	for _, c := range p.idleClients {
		c.OnGoAway()
		if c.keepAlive != nil {
			c.keepAlive.keepAlive.Stop()
		}
	}
}

func (p *poolBinding) newActiveClient(ctx context.Context) (*activeClientBinding, types.PoolFailureReason) {
	connID := getDownstreamConnID(ctx)
	ac := &activeClientBinding{
		protocol:         p.connpool.codec.ProtocolName(),
		downstreamConnID: connID,
		pool:             p,
		host:             p.Host().CreateConnection(ctx),
	}

	host := p.Host()

	connCtx := ctx

	_ = variable.Set(connCtx, types.VariableUpstreamConnectionID, ac.host.Connection.ID())
	_ = variable.Set(connCtx, types.VariableUpstreamProtocol, ac.protocol) // TODO: make sure we need it?

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
	// In the future, methods should be added to determine the protocol capability
	proto := p.connpool.codec.NewXProtocol(ctx)
	if heartbeater, ok := proto.(api.Heartbeater); ok && heartbeater.Trigger(ctx, 0) != nil {
		// create keepalive
		rpcKeepAlive := NewKeepAlive(ac.codecClient, proto, time.Second)
		rpcKeepAlive.StartIdleTimeout()

		ac.SetHeartBeater(rpcKeepAlive)
	}
	////////// codec client

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
type activeClientBinding struct {
	closeWithActiveReq bool
	downstreamConnID   uint64
	goaway             uint32
	protocol           types.ProtocolName
	keepAlive          *keepAliveListener
	pool               *poolBinding
	codecClient        stream.Client
	host               types.CreateConnectionData
	downstreamConn     api.Connection

	downConnOnce sync.Once
}

func (ac *activeClientBinding) addDownConnListenerOnce(ctx context.Context) {
	ac.downConnOnce.Do(func() {
		downstreamConn := getDownstreamConn(ctx)
		ac.downstreamConn = downstreamConn
		ac.downstreamConn.AddConnectionEventListener(downstreamCloseListener{upstreamClient: ac})
	})
}

// Close close the client
//
//	if err == nil, do nothing, the client is never taken away when GetActiveClient
//	if err != nil, error occurred on this client, remove it from pool and close the connection
func (ac *activeClientBinding) Close(err error) {
	if err != nil {
		ac.removeFromPool()
		ac.host.Connection.Close(api.NoFlush, api.LocalClose)
		return
	}
}

// removeFromPool removes this client from connection pool
func (ac *activeClientBinding) removeFromPool() {
	p := ac.pool

	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	delete(p.idleClients, ac.downstreamConnID)
}

// types.ConnectionEventListener
// nolint: dupl
func (ac *activeClientBinding) OnEvent(event api.ConnectionEvent) {
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

	var communicationFailure = false
	switch {
	case event.IsClose():
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

		// RemoteClose when read/write error
		// LocalClose when there is a panic
		// OnReadErrClose when read failed
		ac.removeFromPool()
		communicationFailure = true

	case event == api.ConnectTimeout:
		host.HostStats().UpstreamRequestTimeout.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTimeout.Inc(1)
		if ac.codecClient != nil {
			ac.codecClient.Close()
		}
		communicationFailure = true
	case event == api.ConnectFailed:
		host.HostStats().UpstreamConnectionConFail.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionConFail.Inc(1)
		communicationFailure = true
	default:
		// do nothing
	}

	if communicationFailure && ac.downstreamConn != nil {
		// if connection go away and all requests are handled ,we do not need to close the downstream
		if atomic.LoadUint32(&ac.goaway) == GoAway && ac.codecClient.ActiveRequestsNum() == 0 {
			return
		}

		ac.downstreamConn.Close(api.NoFlush, api.LocalClose)
	}
}

// types.StreamEventListener
var upperStreamGoAway = errors.New("upper stream go away and no active request exist")

func (ac *activeClientBinding) OnDestroyStream() {
	host := ac.pool.Host()
	host.HostStats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().ResourceManager().Requests().Decrease()

	if atomic.LoadUint32(&ac.goaway) == GoAway && ac.codecClient.ActiveRequestsNum() == 0 {
		ac.Close(upperStreamGoAway)
	}
}

func (ac *activeClientBinding) OnResetStream(reason types.StreamResetReason) {
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
func (ac *activeClientBinding) OnGoAway() {
	ac.removeFromPool()
	atomic.StoreUint32(&ac.goaway, GoAway)

	if ac.codecClient.ActiveRequestsNum() == 0 {
		ac.Close(upperStreamGoAway)
	}
}

// SetHeartBeater set the heart beat for an active client
func (ac *activeClientBinding) SetHeartBeater(hb types.KeepAlive) {
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

func getDownstreamConnID(ctx context.Context) uint64 {
	if ctx != nil {
		if val, err := variable.Get(ctx, types.VariableConnectionID); err == nil {
			if code, ok := val.(uint64); ok {
				return code
			}
		}
	}
	return 0
}
