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
package connpool

import (
	"context"
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"sync"
	"sync/atomic"
)

// for xprotocol
const (
	Init = iota
	Connecting
	Connected
)

// RegisterProtoConnPoolFactory register a protocol connection pool factory
func RegisterProtoConnPoolFactory(proto api.Protocol) {
	network.RegisterNewPoolFactory(proto, NewConnPool)
	types.RegisterConnPoolFactory(proto, true)
}

// types.ConnectionPool
// nolint
type connpool struct {
	host      atomic.Value
	tlsHash   *types.HashValue
	clientMux sync.Mutex

	totalClientCount uint64 // total clients
	protocol         api.Protocol

	destroyed uint64
}

// NewConnPool init a connection pool
func NewConnPool(proto api.Protocol, subProto types.ProtocolName, host types.Host) types.ConnectionPool {
	p := &connpool{
		tlsHash:  host.TLSHashValue(),
		protocol: proto,
	}
	p.host.Store(host)

	var res types.ConnectionPool
	switch xprotocol.GetProtocol(subProto).PoolMode() {
	case types.Multiplex:
		res = &connpoolMultiplex{
			connpool:    p,
			idleClients: make(map[api.Protocol][]*activeClientMultiplex),
		}
	case types.PingPong:
		res = &connpoolPingPong{
			connpool:    p,
			idleClients: make(map[api.Protocol][]*activeClientPingPong),
		}
	default: // TCP proxy mode
		res = &connpoolTCP{
			connpool:    p,
			idleClients: make(map[uint64][]*activeClientTCP),
		}
	}

	return res
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

// ----------xprotocol only
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

// ----------xprotocol only
