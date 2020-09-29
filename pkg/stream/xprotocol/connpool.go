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
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"sync/atomic"
)

func init() {
	network.RegisterNewPoolFactory(protocol.Xprotocol, NewConnPool)
	types.RegisterConnPoolFactory(protocol.Xprotocol, true)
}

// for xprotocol
const (
	Init = iota
	Connecting
	Connected
)

// types.ConnectionPool
// nolint
type connpool struct {
	host             atomic.Value
	tlsHash          *types.HashValue
	totalClientCount uint64 // total clients
	protocol         api.Protocol
}

// NewConnPool init a connection pool
func NewConnPool(ctx context.Context, host types.Host) types.ConnectionPool {
	p := &connpool{
		tlsHash:  host.TLSHashValue(),
		protocol: getProtocol(ctx),
	}
	p.host.Store(host)

	switch xprotocol.GetProtocol(getSubProtocol(ctx)).PoolMode() {
	case types.Multiplex:
		return NewPoolMultiplex(p)
	case types.PingPong:
		return NewPoolPingPong(p)
	default:
		return NewPoolBinding(p) // upstream && downstream connection binding proxy mode
	}
}

func (p *connpool) TLSHashValue() *types.HashValue {
	return p.tlsHash
}

func (p *connpool) Protocol() types.ProtocolName {
	return p.protocol
}

func (p *connpool) Host() types.Host {
	h := p.host.Load()
	if host, ok := h.(types.Host); ok {
		return host
	}

	return nil
}

// keepAliveListener is a types.ConnectionEventListener
type keepAliveListener struct {
	keepAlive types.KeepAlive
	conn      api.Connection
}

// OnEvent impl types.ConnectionEventListener
func (l *keepAliveListener) OnEvent(event api.ConnectionEvent) {
	if event == api.OnReadTimeout && l.keepAlive != nil {
		l.keepAlive.SendKeepAlive()
	}
}

func getProtocol(ctx context.Context) types.ProtocolName {
	if ctx != nil {
		if val := mosnctx.Get(ctx, types.ContextKeyConfigUpStreamProtocol); val != nil {
			if code, ok := val.(string); ok {
				return types.ProtocolName(code)
			}
		}
	}
	return ""
}

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

func getDownstreamConn(ctx context.Context) api.Connection {
	if ctx != nil {
		if val := mosnctx.Get(ctx, types.ContextKeyConnection); val != nil {
			if conn, ok := val.(api.Connection); ok {
				return conn
			}
		}
	}
	return nil
}
