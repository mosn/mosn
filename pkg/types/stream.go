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

package types

import (
	"context"
)

//
//   The bunch of interfaces are structure skeleton to build a extensible stream multiplexing architecture. The core concept is mainly refer to golang HTTP2 and envoy.
//
//   In mosn, we have 4 layers to build a mesh, stream is the inheritance layer to bond protocol layer and proxy layer together.
//	 -----------------------
//   |        PROXY          |
//    -----------------------
//   |       STREAMING       |
//    -----------------------
//   |        PROTOCOL       |
//    -----------------------
//   |         NET/IO        |
//    -----------------------
//
//   Core model in stream layer is stream, which manages process of a round-trip, a request and a corresponding response.
// 	 Event listeners can be installed into a stream to monitor event.
//	 Stream has two related models, encoder and decoder:
// 		- StreamSender: a sender encodes request/response to binary and sends it out, flag 'endStream' means data is ready to sendout, no need to wait for further input.
//		- StreamReceiveListener: It's more like a decode listener to get called on a receiver receives binary and decodes to a request/response.
//	 	- Stream does not have a predetermined direction, so StreamSender could be a request encoder as a client or a response encoder as a server. It's just about the scenario, so does StreamReceiveListener.
//
//   Stream:
//      - Encoder
// 			- StreamSender
// 		- Decoder
//			- StreamReceiveListener
//
//   Event listeners:
//		- StreamEventListener: listen stream event: reset, destroy.
//		- StreamConnectionEventListener: listen stream connection event: goaway.
//
//	 In order to meet the expansion requirements in the stream processing, StreamSenderFilter and StreamReceiverFilter are introduced as a filter chain in encode/decode process.
//   Filter's method will be called on corresponding stream process stage and returns a status(Continue/Stop) to effect the control flow.
//
//   From an abstract perspective, stream represents a virtual process on underlying connection. To make stream interactive with connection, some intermediate object can be used.
//	 StreamConnection is the core model to connect connection system to stream system. As a example, when proxy reads binary data from connection, it dispatches data to StreamConnection to do protocol decode.
//   Specifically, ClientStreamConnection uses a NewStream to exchange StreamReceiveListener with StreamSender.
//   Engine provides a callbacks(StreamSenderFilterHandler/StreamReceiverFilterHandler) to let filter interact with stream engine.
// 	 As a example, a encoder filter stopped the encode process, it can continue it by StreamSenderFilterHandler.ContinueSending later. Actually, a filter engine is a encoder/decoder itself.
//
//   Below is the basic relation on stream and connection:
//    --------------------------------------------------------------------------------------------------------------
//   |																												|
//   |                              ConnPool                                                                        |
//   |                                 |1                                                                           |
//   |                                 |                                                                            |
//   |                                 |*                                                                           |
//   |                               Client                                                                         |
//   |                                 |1                                                                           |
//   | 	  EventListener   			   |				StreamEventListener											|
//   |        *|                       |                       |*													|
//   |         |                       |                       |													|
//   |        1|        1    1  	   |1 		1        *     |1													|
// 	 |	    Connection -------- StreamConnection ---------- Stream													|
//   |        1|                   	   |1				   	   |1                                                   |
// 	 |		   |					   |				   	   |                                                    |
//	 |         |                   	   |					   |--------------------------------					|
//   |        *|                   	   |					   |*           	 				|*					|
//   |	 ConnectionFilter    		   |			      StreamSender      		        StreamReceiveListener	        |
//   |								   |*					   |1				 				|1					|
// 	 |						StreamConnectionEventListener	   |				 				|					|
//	 |													       |*				 				|*					|
//	 |										 	 		StreamSenderFilter	   			StreamReceiverFilter	    |
//	 |													   	   |1								|1					|
//	 |													   	   |								|					|
// 	 |													       |1								|1					|
//	 |										 		StreamSenderFilterHandler     StreamReceiverFilterHandler	    |
//   |																												|
//    --------------------------------------------------------------------------------------------------------------
//

// StreamResetReason defines the reason why stream reset
type StreamResetReason string

// Group of stream reset reasons
const (
	StreamConnectionTermination StreamResetReason = "ConnectionTermination"
	StreamConnectionFailed      StreamResetReason = "ConnectionFailed"
	StreamLocalReset            StreamResetReason = "StreamLocalReset"
	StreamOverflow              StreamResetReason = "StreamOverflow"
	StreamRemoteReset           StreamResetReason = "StreamRemoteReset"
	UpstreamReset               StreamResetReason = "UpstreamReset"
	UpstreamGlobalTimeout       StreamResetReason = "UpstreamGlobalTimeout"
	UpstreamPerTryTimeout       StreamResetReason = "UpstreamPerTryTimeout"
)

// Stream is a generic protocol stream, it is the core model in stream layer
type Stream interface {
	// ID returns unique stream id during one connection life-cycle
	ID() uint64

	// AddEventListener adds stream event listener
	AddEventListener(streamEventListener StreamEventListener)

	// RemoveEventListener removes stream event listener
	RemoveEventListener(streamEventListener StreamEventListener)

	// ResetStream rests and destroys stream, called on exception cases like connection close.
	// Any registered StreamEventListener.OnResetStream and OnDestroyStream will be called.
	ResetStream(reason StreamResetReason)

	// DestroyStream destroys stream, called after stream process in client/server cases.
	// Any registered StreamEventListener.OnDestroyStream will be called.
	DestroyStream()
}

// StreamEventListener is a stream event listener
type StreamEventListener interface {
	// OnResetStream is called on a stream is been reset
	OnResetStream(reason StreamResetReason)

	// OnDestroyStream is called on stream destroy
	OnDestroyStream()
}

// StreamSender encodes and sends protocol stream
// On server scenario, StreamSender sends response
// On client scenario, StreamSender sends request
type StreamSender interface {
	// Append headers
	// endStream supplies whether this is a header only request/response
	AppendHeaders(ctx context.Context, headers HeaderMap, endStream bool) error

	// Append data
	// endStream supplies whether this is the last data frame
	AppendData(ctx context.Context, data IoBuffer, endStream bool) error

	// Append trailers, implicitly ends the stream.
	AppendTrailers(ctx context.Context, trailers HeaderMap) error

	// Get related stream
	GetStream() Stream
}

// StreamReceiveListener is called on data received and decoded
// On server scenario, StreamReceiveListener is called to handle request
// On client scenario, StreamReceiveListener is called to handle response
type StreamReceiveListener interface {
	// OnReceiveHeaders is called with decoded headers
	// endStream supplies whether this is a header only request/response
	OnReceiveHeaders(ctx context.Context, headers HeaderMap, endOfStream bool)

	// OnReceiveData is called with a decoded data
	// endStream supplies whether this is the last data
	OnReceiveData(ctx context.Context, data IoBuffer, endOfStream bool)

	// OnReceiveTrailers is called with a decoded trailers frame, implicitly ends the stream.
	OnReceiveTrailers(ctx context.Context, trailers HeaderMap)

	// OnDecodeError is called with when exception occurs
	OnDecodeError(ctx context.Context, err error, headers HeaderMap)

	OnReceive(ctx context.Context, headers HeaderMap, data IoBuffer, trailers HeaderMap)
}

// StreamConnection is a connection runs multiple streams
type StreamConnection interface {
	// Dispatch incoming data
	// On data read scenario, it connects connection and stream by dispatching read buffer to stream,
	// stream uses protocol decode data, and popup event to controller
	Dispatch(buffer IoBuffer)

	// Protocol on the connection
	Protocol() Protocol

	// Active streams count
	ActiveStreamsNum() int

	// GoAway sends go away to remote for graceful shutdown
	GoAway()

	// Reset underlying streams
	Reset(reason StreamResetReason)
}

// ServerStreamConnection is a server side stream connection.
type ServerStreamConnection interface {
	StreamConnection
}

// ClientStreamConnection is a client side stream connection.
type ClientStreamConnection interface {
	StreamConnection

	// NewStream starts to create a new outgoing request stream and returns a sender to write data
	// responseReceiveListener supplies the response listener on decode event
	// StreamSender supplies the sender to write request data
	NewStream(ctx context.Context, responseReceiveListener StreamReceiveListener) StreamSender
}

// StreamConnectionEventListener is a stream connection event listener
type StreamConnectionEventListener interface {
	// OnGoAway is called on remote sends 'go away'
	OnGoAway()
}

// ServerStreamConnectionEventListener is a stream connection event listener for server connection
type ServerStreamConnectionEventListener interface {
	StreamConnectionEventListener

	// NewStreamDetect returns stream event receiver
	NewStreamDetect(context context.Context, sender StreamSender, spanBuilder SpanBuilder) StreamReceiveListener
}

type StreamFilterBase interface {
	OnDestroy()
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

// StreamSenderFilter is a stream sender filter
type StreamSenderFilter interface {
	StreamFilterBase

	// Append encodes request/response
	Append(ctx context.Context, headers HeaderMap, buf IoBuffer, trailers HeaderMap) StreamFilterStatus

	// SetSenderFilterHandler sets the StreamSenderFilterHandler
	SetSenderFilterHandler(handler StreamSenderFilterHandler)
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

// StreamReceiverFilter is a StreamFilterBase wrapper
type StreamReceiverFilter interface {
	StreamFilterBase

	// OnReceive is called with decoded request/response
	OnReceive(ctx context.Context, headers HeaderMap, buf IoBuffer, trailers HeaderMap) StreamFilterStatus

	// SetReceiveFilterHandler sets decoder filter callbacks
	SetReceiveFilterHandler(handler StreamReceiverFilterHandler)
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

	// TODO: remove all of the following when proxy changed to single request @lieyuan
	// StreamFilters will modified headers/data/trailer in different steps
	// for example, maybe modify headers in on receive data
	GetRequestHeaders() HeaderMap
	SetRequestHeaders(headers HeaderMap)

	GetRequestData() IoBuffer
	SetRequestData(buf IoBuffer)

	GetRequestTrailers() HeaderMap
	SetRequestTrailers(trailers HeaderMap)
}

// StreamFilterChainFactory adds filter into callbacks
type StreamFilterChainFactory interface {
	CreateFilterChain(context context.Context, callbacks StreamFilterChainFactoryCallbacks)
}

// StreamFilterChainFactoryCallbacks is called in StreamFilterChainFactory
type StreamFilterChainFactoryCallbacks interface {
	AddStreamSenderFilter(filter StreamSenderFilter)

	AddStreamReceiverFilter(filter StreamReceiverFilter, p Phase)

	// add access log per stream
	AddStreamAccessLog(accessLog AccessLog)
}

type StreamFilterStatus string

// StreamFilterStatus types
const (
	// Continue filter chain iteration.
	StreamFilterContinue StreamFilterStatus = "Continue"
	// Do not iterate to next iterator.
	StreamFilterStop StreamFilterStatus = "Stop"

	StreamFilterReMatchRoute StreamFilterStatus = "Retry Match Route"
)

// PoolFailureReason type
type PoolFailureReason string

// PoolFailureReason types
const (
	Overflow          PoolFailureReason = "Overflow"
	ConnectionFailure PoolFailureReason = "ConnectionFailure"
)

//  ConnectionPool is a connection pool interface to extend various of protocols
type ConnectionPool interface {
	Protocol() Protocol

	NewStream(ctx context.Context, receiver StreamReceiveListener, listener PoolEventListener)

	Close()
}

type PoolEventListener interface {
	OnFailure(reason PoolFailureReason, host Host)

	OnReady(sender StreamSender, host Host)
}
