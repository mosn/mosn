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

package stream

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/types"
)

type CodecClient interface {
	types.ConnectionEventListener
	types.ReadFilter

	ID() uint64

	AddConnectionCallbacks(cb types.ConnectionEventListener)

	ActiveRequestsNum() int

	NewStream(context context.Context, streamID string, respDecoder types.StreamReceiver) types.StreamSender

	SetConnectionStats(stats *types.ConnectionStats)

	SetCodecClientCallbacks(cb CodecClientCallbacks)

	SetCodecConnectionCallbacks(cb types.StreamConnectionEventListener)

	Close()

	RemoteClose() bool
}

type CodecClientCallbacks interface {
	OnStreamDestroy()

	OnStreamReset(reason types.StreamResetReason)
}

type ProtocolStreamFactory interface {
	CreateClientStream(context context.Context, connection types.ClientConnection,
		streamConnCallbacks types.StreamConnectionEventListener,
		callbacks types.ConnectionEventListener) types.ClientStreamConnection

	CreateServerStream(context context.Context, connection types.Connection,
		callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection

	CreateBiDirectStream(context context.Context, connection types.ClientConnection,
		clientCallbacks types.StreamConnectionEventListener,
		serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection
}
