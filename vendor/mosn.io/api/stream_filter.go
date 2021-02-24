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

package api

import (
	"context"
)

type StreamFilterStatus string

// StreamFilterStatus types
const (
	// Continue filter chain iteration.
	StreamFilterContinue StreamFilterStatus = "Continue"
	// Do not iterate to next iterator.
	StreamFilterStop StreamFilterStatus = "Stop"
	// terminate request.
	StreamFiltertermination StreamFilterStatus = "termination"

	StreamFilterReMatchRoute StreamFilterStatus = "Retry Match Route"
	StreamFilterReChooseHost StreamFilterStatus = "Retry Choose Host"
)

type StreamFilterBase interface {
	OnDestroy()
}

// StreamSenderFilter is a stream sender filter
type StreamSenderFilter interface {
	StreamFilterBase

	// Append encodes request/response
	Append(ctx context.Context, headers HeaderMap, buf IoBuffer, trailers HeaderMap) StreamFilterStatus

	// SetSenderFilterHandler sets the StreamSenderFilterHandler
	SetSenderFilterHandler(handler StreamSenderFilterHandler)
}

// StreamReceiverFilter is a StreamFilterBase wrapper
type StreamReceiverFilter interface {
	StreamFilterBase

	// OnReceive is called with decoded request/response
	OnReceive(ctx context.Context, headers HeaderMap, buf IoBuffer, trailers HeaderMap) StreamFilterStatus

	// SetReceiveFilterHandler sets decoder filter callbacks
	SetReceiveFilterHandler(handler StreamReceiverFilterHandler)
}

// StreamFilterHandler is called by stream filter to interact with underlying stream
type StreamFilterHandler interface {
	// Route returns a route for current stream
	Route() Route

	// RequestInfo returns request info related to the stream
	RequestInfo() RequestInfo

	// Connection returns the originating connection
	Connection() Connection
}

// StreamSenderFilterHandler is a StreamFilterHandler wrapper
type StreamSenderFilterHandler interface {
	StreamFilterHandler

	// TODO :remove all of the following when proxy changed to single request @lieyuan
	// StreamFilters will modified headers/data/trailer in different steps
	// for example, maybe modify headers in AppendData
	GetResponseHeaders() HeaderMap
	SetResponseHeaders(headers HeaderMap)

	GetResponseData() IoBuffer
	SetResponseData(buf IoBuffer)

	GetResponseTrailers() HeaderMap
	SetResponseTrailers(trailers HeaderMap)
}

// StreamReceiverFilterHandler add additional callbacks that allow a decoding filter to restart
// decoding if they decide to hold data
type StreamReceiverFilterHandler interface {
	StreamFilterHandler

	// TODO: consider receiver filter needs AppendXXX or not

	// AppendHeaders is called with headers to be encoded, optionally indicating end of stream
	// Filter uses this function to send out request/response headers of the stream
	// endStream supplies whether this is a header only request/response
	AppendHeaders(headers HeaderMap, endStream bool)

	// AppendData is called with data to be encoded, optionally indicating end of stream.
	// Filter uses this function to send out request/response data of the stream
	// endStream supplies whether this is the last data
	AppendData(buf IoBuffer, endStream bool)

	// AppendTrailers is called with trailers to be encoded, implicitly ends the stream.
	// Filter uses this function to send out request/response trailers of the stream
	AppendTrailers(trailers HeaderMap)

	// SendHijackReply is called when the filter will response directly
	SendHijackReply(code int, headers HeaderMap)

	// SendHijackReplyWithBody is called when the filter will response directly with body
	SendHijackReplyWithBody(code int, headers HeaderMap, body string)

	// SendDirectRespoonse is call when the filter will response directly
	SendDirectResponse(headers HeaderMap, buf IoBuffer, trailers HeaderMap)

	// TerminateStream can force terminate a request asynchronously.
	// The response status code should be HTTP status code.
	// If the request is already finished, returns false.
	TerminateStream(code int) bool

	// TODO: remove all of the following when proxy changed to single request @lieyuan
	// StreamFilters will modified headers/data/trailer in different steps
	// for example, maybe modify headers in on receive data
	GetRequestHeaders() HeaderMap
	SetRequestHeaders(headers HeaderMap)

	GetRequestData() IoBuffer
	SetRequestData(buf IoBuffer)

	GetRequestTrailers() HeaderMap
	SetRequestTrailers(trailers HeaderMap)

	SetConvert(on bool)

	// GetFilterCurrentPhase get current phase for filter
	GetFilterCurrentPhase() ReceiverFilterPhase
}

// StreamFilterChainFactory adds filter into callbacks
type StreamFilterChainFactory interface {
	CreateFilterChain(context context.Context, callbacks StreamFilterChainFactoryCallbacks)
}

// StreamFilterChainFactoryCallbacks is called in StreamFilterChainFactory
type StreamFilterChainFactoryCallbacks interface {
	AddStreamSenderFilter(filter StreamSenderFilter, p SenderFilterPhase)

	AddStreamReceiverFilter(filter StreamReceiverFilter, p ReceiverFilterPhase)

	// add access log per stream
	AddStreamAccessLog(accessLog AccessLog)
}

type ReceiverFilterPhase int

const (
	BeforeRoute ReceiverFilterPhase = iota
	AfterRoute
	AfterChooseHost
)

type SenderFilterPhase int

const (
	BeforeSend SenderFilterPhase = iota
)
