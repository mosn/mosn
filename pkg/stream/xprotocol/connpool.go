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
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

// for xprotocol
const (
	Init = iota
	Connecting
	Connected
	GoAway // received GoAway frame
)

// types.ConnectionPool
// nolint
type connpool struct {
	host     atomic.Value
	tlsHash  *types.HashValue
	protocol api.ProtocolName
	codec    api.XProtocolCodec
}

// NewConnPool init a connection pool
func NewConnPool(ctx context.Context, codec api.XProtocolCodec, host types.Host) types.ConnectionPool {
	proto := codec.NewXProtocol(ctx)
	p := &connpool{
		tlsHash:  host.TLSHashValue(),
		protocol: proto.Name(),
		codec:    codec,
	}
	p.host.Store(host)

	switch proto.PoolMode() {
	case api.Multiplex:
		return NewPoolMultiplex(p)
	case api.PingPong:
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

func getDownstreamConn(ctx context.Context) api.Connection {
	if ctx != nil {
		if val, err := variable.Get(ctx, types.VariableConnection); err == nil && val != nil {
			if conn, ok := val.(api.Connection); ok {
				return conn
			}
		}
	}
	return nil
}
