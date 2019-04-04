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

// stream.Client
// types.ReadFilter
// types.StreamConnectionEventListener
type client struct {
	Protocol                      types.Protocol
	Connection                    types.ClientConnection
	Host                          types.HostInfo
	ClientStreamConnection        types.ClientStreamConnection
	StreamConnectionEventListener types.StreamConnectionEventListener
	ConnectedFlag                 bool
}

// NewStreamClient
// Create a codecclient used as a client to send/receive stream in a connection
func NewStreamClient(ctx context.Context, prot types.Protocol, connection types.ClientConnection, host types.HostInfo) Client {
	client := &client{
		Protocol:   prot,
		Connection: connection,
		Host:       host,
	}

	if factory, ok := streamFactories[prot]; ok {
		client.ClientStreamConnection = factory.CreateClientStream(ctx, connection, client, client)
	} else {
		return nil
	}

	connection.AddConnectionEventListener(client)
	connection.FilterManager().AddReadFilter(client)
	connection.SetNoDelay(true)

	return client
}

// NewBiDirectStreamClient
// Create a bidirectional client used to realize bidirectional communication
func NewBiDirectStreamClient(ctx context.Context, prot types.Protocol, connection types.ClientConnection, host types.HostInfo,
	serverCallbacks types.ServerStreamConnectionEventListener) Client {
	client := &client{
		Protocol:   prot,
		Connection: connection,
		Host:       host,
	}

	if factory, ok := streamFactories[prot]; ok {
		client.ClientStreamConnection = factory.CreateBiDirectStream(ctx, connection, client, serverCallbacks)
	} else {
		return nil
	}

	connection.AddConnectionEventListener(client)
	connection.FilterManager().AddReadFilter(client)
	connection.SetNoDelay(true)

	return client
}

// Client
func (c *client) ConnID() uint64 {
	return c.Connection.ID()
}

func (c *client) Connect(ioEnabled bool) error {
	return c.Connection.Connect(ioEnabled)
}

func (c *client) AddConnectionEventListener(listener types.ConnectionEventListener) {
	c.Connection.AddConnectionEventListener(listener)
}

func (c *client) ActiveRequestsNum() int {
	return c.ClientStreamConnection.ActiveStreamsNum()
}

func (c *client) SetConnectionStats(stats *types.ConnectionStats) {
	c.Connection.SetStats(stats)
}

func (c *client) SetStreamConnectionEventListener(listener types.StreamConnectionEventListener) {
	c.StreamConnectionEventListener = listener
}

func (c *client) NewStream(context context.Context, respReceiver types.StreamReceiveListener) types.StreamSender {
	wrapper := &clientStreamReceiverWrapper{
		streamReceiver: respReceiver,
	}

	streamSender := c.ClientStreamConnection.NewStream(context, wrapper)
	wrapper.stream = streamSender.GetStream()

	return streamSender
}

func (c *client) Close() {
	c.Connection.Close(types.NoFlush, types.LocalClose)
}

// types.StreamConnectionEventListener
func (c *client) OnGoAway() {
	c.StreamConnectionEventListener.OnGoAway()
}

// types.ConnectionEventListener
// conn callbacks
func (c *client) OnEvent(event types.ConnectionEvent) {
	switch event {
	case types.Connected:
		c.ConnectedFlag = true
	}

	if event.IsClose() || event.ConnectFailure() {
		reason := types.StreamConnectionFailed

		if c.ConnectedFlag {
			reason = types.StreamConnectionTermination
		}

		c.ClientStreamConnection.Reset(reason)
	}
}

// types.ReadFilter
// read filter, recv upstream data
func (c *client) OnData(buffer types.IoBuffer) types.FilterStatus {
	c.ClientStreamConnection.Dispatch(buffer)

	return types.Stop
}

func (c *client) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (c *client) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}

// uniform wrapper to destroy stream at client side
type clientStreamReceiverWrapper struct {
	stream         types.Stream
	streamReceiver types.StreamReceiveListener
}

func (w *clientStreamReceiverWrapper) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	w.stream.DestroyStream()
	w.streamReceiver.OnReceive(ctx, headers, data, trailers)
}

func (w *clientStreamReceiverWrapper) OnReceiveHeaders(ctx context.Context, headers types.HeaderMap, endOfStream bool) {
	if endOfStream {
		w.stream.DestroyStream()
	}

	w.streamReceiver.OnReceiveHeaders(ctx, headers, endOfStream)
}

func (w *clientStreamReceiverWrapper) OnReceiveData(ctx context.Context, data types.IoBuffer, endOfStream bool) {
	if endOfStream {
		w.stream.DestroyStream()
	}

	w.streamReceiver.OnReceiveData(ctx, data, endOfStream)
}

func (w *clientStreamReceiverWrapper) OnReceiveTrailers(ctx context.Context, trailers types.HeaderMap) {
	w.stream.DestroyStream()

	w.streamReceiver.OnReceiveTrailers(ctx, trailers)
}

func (w *clientStreamReceiverWrapper) OnDecodeError(ctx context.Context, err error, headers types.HeaderMap) {
	w.streamReceiver.OnDecodeError(ctx, err, headers)
}
