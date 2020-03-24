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

	metrics "github.com/rcrowley/go-metrics"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

// stream.Client
// types.ReadFilter
// types.StreamConnectionEventListener
type client struct {
	Protocol                      types.ProtocolName
	Connection                    types.ClientConnection
	Host                          types.Host
	ClientStreamConnection        types.ClientStreamConnection
	StreamConnectionEventListener types.StreamConnectionEventListener
	ConnectedFlag                 bool
}

// NewStreamClient
// Create a codecclient used as a client to send/receive stream in a connection
func NewStreamClient(ctx context.Context, prot api.Protocol, connection types.ClientConnection, host types.Host) Client {
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
func NewBiDirectStreamClient(ctx context.Context, prot api.Protocol, connection types.ClientConnection, host types.Host,
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

func (c *client) Connect() error {
	return c.Connection.Connect()
}

func (c *client) AddConnectionEventListener(listener api.ConnectionEventListener) {
	c.Connection.AddConnectionEventListener(listener)
}

func (c *client) ActiveRequestsNum() int {
	return c.ClientStreamConnection.ActiveStreamsNum()
}

func (c *client) SetConnectionCollector(read, write metrics.Counter) {
	c.Connection.SetCollector(read, write)
}

func (c *client) SetStreamConnectionEventListener(listener types.StreamConnectionEventListener) {
	c.StreamConnectionEventListener = listener
}

func (c *client) NewStream(context context.Context, respReceiver types.StreamReceiveListener) types.StreamSender {
	// oneway
	if respReceiver == nil {
		log.DefaultLogger.Debugf("oneway client NewStream")
		return c.ClientStreamConnection.NewStream(context, nil)
	}

	wrapper := &clientStreamReceiverWrapper{
		streamReceiver: respReceiver,
	}

	streamSender := c.ClientStreamConnection.NewStream(context, wrapper)
	wrapper.stream = streamSender.GetStream()

	return streamSender
}

func (c *client) Close() {
	c.Connection.Close(api.NoFlush, api.LocalClose)
}

// types.StreamConnectionEventListener
func (c *client) OnGoAway() {
	c.StreamConnectionEventListener.OnGoAway()
}

// types.ConnectionEventListener
// conn callbacks
func (c *client) OnEvent(event api.ConnectionEvent) {
	log.DefaultLogger.Debugf("client OnEvent %v, connected %v", event, c.ConnectedFlag)
	switch event {
	case api.Connected:
		c.ConnectedFlag = true
	}

	if reason, ok := c.ClientStreamConnection.CheckReasonError(c.ConnectedFlag, event); !ok {
		c.ClientStreamConnection.Reset(reason)
	}
}

// types.ReadFilter
// read filter, recv upstream data
func (c *client) OnData(buffer buffer.IoBuffer) api.FilterStatus {
	c.ClientStreamConnection.Dispatch(buffer)

	return api.Stop
}

func (c *client) OnNewConnection() api.FilterStatus {
	return api.Continue
}

func (c *client) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {}

// uniform wrapper to destroy stream at client side
type clientStreamReceiverWrapper struct {
	stream         types.Stream
	streamReceiver types.StreamReceiveListener
}

func (w *clientStreamReceiverWrapper) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	w.stream.DestroyStream()
	w.streamReceiver.OnReceive(ctx, headers, data, trailers)
}

func (w *clientStreamReceiverWrapper) OnDecodeError(ctx context.Context, err error, headers types.HeaderMap) {
	w.streamReceiver.OnDecodeError(ctx, err, headers)
}
