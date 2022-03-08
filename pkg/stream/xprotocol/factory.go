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
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
)

type streamConnFactory struct {
	name    api.ProtocolName
	matcher api.ProtocolMatch
	factory func(ctx context.Context) api.XProtocol
}

func NewStreamFactory(codec api.XProtocolCodec) types.ProtocolStreamFactory {
	return &streamConnFactory{
		name:    codec.ProtocolName(),
		matcher: codec.ProtocolMatch(),
		factory: codec.NewXProtocol,
	}
}

func (f *streamConnFactory) CreateClientStream(context context.Context, connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener, connCallbacks api.ConnectionEventListener) types.ClientStreamConnection {
	return f.newStreamConnection(context, connection, clientCallbacks, nil).(types.ClientStreamConnection)
}

func (f *streamConnFactory) CreateServerStream(context context.Context, connection api.Connection,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	return f.newStreamConnection(context, connection, nil, serverCallbacks).(types.ServerStreamConnection)
}

func (f *streamConnFactory) CreateBiDirectStream(context context.Context, connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {
	return f.newStreamConnection(context, connection, clientCallbacks, serverCallbacks).(types.ClientStreamConnection)
}

func (f *streamConnFactory) ProtocolMatch(context context.Context, prot string, magic []byte) error {
	// if matcher is nil, means protocol does not support for multiple protocol mode and auto mode.
	if f.matcher == nil {
		return stream.FAILED
	}

	result := f.matcher(magic)
	switch result {
	case api.MatchSuccess:
		return nil
	case api.MatchAgain:
		return stream.EAGAIN
	}
	return stream.FAILED

}
