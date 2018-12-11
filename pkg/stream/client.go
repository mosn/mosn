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
	"container/list"
	"context"
	"sync"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/types"
)

// stream.Client
// types.ReadFilter
// types.StreamConnectionEventListener
type streamClient struct {
	Protocol                 types.Protocol
	Connection               types.ClientConnection
	Host                     types.HostInfo
	ClientStreamConn         types.ClientStreamConnection
	ActiveRequests           *list.List
	AcrMux                   sync.RWMutex
	ClientListener           ClientListener
	StreamConnectionListener types.StreamConnectionEventListener
	ConnectedFlag            bool
}

// NewStreamClient
// Create a codecclient used as a client to send/receive stream in a connection
func NewStreamClient(ctx context.Context, prot types.Protocol, connection types.ClientConnection, host types.HostInfo) Client {
	client := &streamClient{
		Protocol:       prot,
		Connection:     connection,
		Host:           host,
		ActiveRequests: list.New(),
	}

	if factory, ok := streamFactories[prot]; ok {
		client.ClientStreamConn = factory.CreateClientStream(ctx, connection, client, client)
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
	client := &streamClient{
		Protocol:       prot,
		Connection:     connection,
		Host:           host,
		ActiveRequests: list.New(),
	}

	if factory, ok := streamFactories[prot]; ok {
		client.ClientStreamConn = factory.CreateBiDirectStream(ctx, connection, client, serverCallbacks)
	} else {
		return nil
	}

	connection.AddConnectionEventListener(client)
	connection.FilterManager().AddReadFilter(client)
	connection.SetNoDelay(true)

	return client
}

func (c *streamClient) ConnID() uint64 {
	return c.Connection.ID()
}

func (c *streamClient) AddConnectionEventListener(listener types.ConnectionEventListener) {
	c.Connection.AddConnectionEventListener(listener)
}

func (c *streamClient) ActiveRequestsNum() int {
	c.AcrMux.RLock()
	defer c.AcrMux.RUnlock()

	return c.ActiveRequests.Len()
}

func (c *streamClient) SetConnectionStats(stats *types.ConnectionStats) {
	c.Connection.SetStats(stats)
}

func (c *streamClient) SetClientListener(listener ClientListener) {
	c.ClientListener = listener
}

func (c *streamClient) SetConnectionEventListener(listener types.StreamConnectionEventListener) {
	c.StreamConnectionListener = listener
}

func (c *streamClient) NewStream(context context.Context, respDecoder types.StreamReceiver) types.StreamSender {
	ar := newActiveRequest(c, respDecoder)
	ar.requestSender = c.ClientStreamConn.NewStream(context, ar)
	ar.requestSender.GetStream().AddEventListener(ar)

	c.AcrMux.Lock()
	ar.element = c.ActiveRequests.PushBack(ar)
	c.AcrMux.Unlock()

	return ar.requestSender
}

func (c *streamClient) Close() {
	c.Connection.Close(types.NoFlush, types.LocalClose)
}

// types.StreamConnectionEventListener
func (c *streamClient) OnGoAway() {
	c.StreamConnectionListener.OnGoAway()
}

// conn callbacks
func (c *streamClient) OnEvent(event types.ConnectionEvent) {
	switch event {
	case types.Connected:
		c.ConnectedFlag = true
	}

	if event.IsClose() || event.ConnectFailure() {
		var arNext *list.Element

		c.AcrMux.RLock()
		acReqs := make([]*activeRequest, 0, c.ActiveRequests.Len())
		for ar := c.ActiveRequests.Front(); ar != nil; ar = arNext {
			arNext = ar.Next()
			acReqs = append(acReqs, ar.Value.(*activeRequest))
		}
		c.AcrMux.RUnlock()

		for _, ac := range acReqs {
			reason := types.StreamConnectionFailed

			if c.ConnectedFlag {
				reason = types.StreamConnectionTermination
			}
			ac.requestSender.GetStream().ResetStream(reason)
		}
	}
}

// read filter, recv upstream data
func (c *streamClient) OnData(buffer types.IoBuffer) types.FilterStatus {
	c.ClientStreamConn.Dispatch(buffer)

	return types.Stop
}

func (c *streamClient) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (c *streamClient) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}

func (c *streamClient) onReset(request *activeRequest, reason types.StreamResetReason) {
	if c.ClientListener != nil {
		c.ClientListener.OnStreamReset(reason)
	}

	c.deleteRequest(request)
}

func (c *streamClient) responseDecodeComplete(request *activeRequest) {
	c.deleteRequest(request)
	request.requestSender.GetStream().RemoveEventListener(request)
}

func (c *streamClient) deleteRequest(request *activeRequest) {
	if !atomic.CompareAndSwapUint32(&request.deleted, 0, 1) {
		return
	}

	c.AcrMux.Lock()
	defer c.AcrMux.Unlock()

	c.ActiveRequests.Remove(request.element)

	if c.ClientListener != nil {
		c.ClientListener.OnStreamDestroy()
	}
}

// types.StreamEventListener
// types.StreamDecoderWrapper
type activeRequest struct {
	client           *streamClient
	responseReceiver types.StreamReceiver
	requestSender    types.StreamSender
	element          *list.Element
	deleted          uint32
}

func newActiveRequest(codecClient *streamClient, streamDecoder types.StreamReceiver) *activeRequest {
	return &activeRequest{
		client:           codecClient,
		responseReceiver: streamDecoder,
	}
}

func (r *activeRequest) OnResetStream(reason types.StreamResetReason) {
	r.client.onReset(r, reason)
}

func (r *activeRequest) OnReceiveHeaders(context context.Context, headers types.HeaderMap, endStream bool) {
	if endStream {
		r.onPreDecodeComplete()
	}

	r.responseReceiver.OnReceiveHeaders(context, headers, endStream)

	if endStream {
		r.onDecodeComplete()
	}
}

func (r *activeRequest) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {
	if endStream {
		r.onPreDecodeComplete()
	}

	r.responseReceiver.OnReceiveData(context, data, endStream)

	if endStream {
		r.onDecodeComplete()
	}
}

func (r *activeRequest) OnReceiveTrailers(context context.Context, trailers types.HeaderMap) {
	r.onPreDecodeComplete()
	r.responseReceiver.OnReceiveTrailers(context, trailers)
	r.onDecodeComplete()
}

func (r *activeRequest) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
}

func (r *activeRequest) onPreDecodeComplete() {
	r.client.responseDecodeComplete(r)
}

func (r *activeRequest) onDecodeComplete() {}
