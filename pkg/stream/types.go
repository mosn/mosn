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
	"errors"

	metrics "github.com/rcrowley/go-metrics"
	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

// StreamDirection represent the stream's direction
type StreamDirection int

// ServerStream = 1
// ClientStream = 0
const (
	ServerStream StreamDirection = 1
	ClientStream StreamDirection = 0
)

var FAILED = errors.New("FAILED")
var EAGAIN = errors.New("AGAIN")

type Client interface {
	api.ConnectionEventListener
	api.ReadFilter

	ConnID() uint64

	Connect() error

	ActiveRequestsNum() int

	NewStream(context context.Context, respDecoder types.StreamReceiveListener) types.StreamSender

	SetConnectionCollector(read, write metrics.Counter)

	AddConnectionEventListener(listener api.ConnectionEventListener)

	SetStreamConnectionEventListener(listener types.StreamConnectionEventListener)

	Close()
}

type ProtocolStreamFactory interface {
	CreateClientStream(context context.Context, connection types.ClientConnection,
		streamConnCallbacks types.StreamConnectionEventListener,
		callbacks api.ConnectionEventListener) types.ClientStreamConnection

	CreateServerStream(context context.Context, connection api.Connection,
		callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection

	CreateBiDirectStream(context context.Context, connection types.ClientConnection,
		clientCallbacks types.StreamConnectionEventListener,
		serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection

	ProtocolMatch(context context.Context, prot string, magic []byte) error
}
