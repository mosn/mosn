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

import "context"

//
//   The bunch of interfaces are structure skeleton to build a extensible protocol stream architecture.
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
//		- StreamReceiver: It's more like a decode listener to get called on a receiver receives binary and decodes to a request/response.
//	 	- Stream does not have a predetermined direction, so StreamSender could be a request encoder as a client or a response encoder as a server. It's just about the scenario, so does StreamReceiver.
//
//   Stream:
//   	- Event listener
// 			- StreamEventListener
//      - Encoder
// 			- StreamSender
// 		- Decoder
//			- StreamReceiver
//
//	 In order to meet the expansion requirements in the stream processing, StreamEncoderFilters and StreamDecoderFilters are introduced as a filter chain in encode/decode process.
//   Filter's method will be called on corresponding stream process stage and returns a status(Continue/Stop) to effect the control flow.
//
//   From an abstract perspective, stream represents a virtual process on underlying connection. To make stream interactive with connection, some intermediate object can be used.
//	 StreamConnection is the core model to connect connection system to stream system. As a example, when proxy reads binary data from connection, it dispatches data to StreamConnection to do protocol decode.
//   Specifically, ClientStreamConnection uses a NewStream to exchange StreamReceiver with StreamSender.
//   Engine provides a callbacks(StreamSenderFilterCallbacks/StreamReceiverFilterCallbacks) to let filter interact with stream engine.
// 	 As a example, a encoder filter stopped the encode process, it can continue it by StreamSenderFilterCallbacks.ContinueEncoding later. Actually, a filter engine is a encoder/decoder itself.
//
//   Below is the basic relation on stream and connection:
//    --------------------------------------------------------------------------------------------------------------
//   |																												|
//   | 	  EventListener   								EventListener												|
//   |        *|                                               |*													|
//   |         |                                               |													|
//   |        1|        1    1  	    		1     *        |1													|
// 	 |	    Connection -------- StreamConnection ---------- Stream													|
//   |        1|                   	   |1				   	   |1                                                   |
// 	 |		   |					   |				   	   |                                                    |
//	 |         |                   	   |					   |--------------------------------					|
//   |        *|                   	   |					   |*           	 				|*					|
//   |	 ConnectionFilter    		   |			    StreamSender[sender]  		     StreamReceiver[receiver]	|
//   |								   |*					   |1				 				|1					|
// 	 |						StreamConnectionEventListener	   |				 				|					|
//	 |													       |*				 				|*					|
//	 |										 	 		StreamReceiverFilter	   			StreamReceiverFilter	|
//	 |													   	   |1								|1					|
//	 |													   	   |								|					|
// 	 |													       |1								|1					|
//	 |										 		StreamReceiverFilterCallbacks     StreamReceiverFilterCallbacks	|
//   |																												|
//    --------------------------------------------------------------------------------------------------------------
//

type StreamResetReason string

const (
	StreamConnectionTermination StreamResetReason = "ConnectionTermination"
	StreamConnectionFailed      StreamResetReason = "ConnectionFailed"
	StreamLocalReset            StreamResetReason = "StreamLocalReset"
	StreamOverflow              StreamResetReason = "StreamOverflow"
	StreamRemoteReset           StreamResetReason = "StreamRemoteReset"
)

// Core model in stream layer, a generic protocol stream
type Stream interface {
	// Add stream event listener
	AddEventListener(streamEventListener StreamEventListener)

	// Remove stream event listener
	RemoveEventListener(streamEventListener StreamEventListener)

	// Reset stream. Any registered StreamEventListener.OnResetStream should be called.
	ResetStream(reason StreamResetReason)

	// Enable/disable further stream data
	ReadDisable(disable bool)
}

// Stream event listener
type StreamEventListener interface {
	// Called on a stream is been reset
	OnResetStream(reason StreamResetReason)
}

// StreamSender encodes protocol stream
// On server scenario, StreamSender handles response
// On client scenario, StreamSender handles request
type StreamSender interface {
	// Encode headers
	// endStream supplies whether this is a header only request/response
	AppendHeaders(headers interface{}, endStream bool) error

	// Encode data
	// endStream supplies whether this is the last data frame
	AppendData(data IoBuffer, endStream bool) error

	// Encode trailers, implicitly ends the stream.
	AppendTrailers(trailers map[string]string) error

	// Get related stream
	GetStream() Stream
}

// Listeners called on decode stream event
// On server scenario, StreamReceiver handles request
// On client scenario, StreamSender handles response
type StreamReceiver interface {
	// Called with decoded headers
	// endStream supplies whether this is a header only request/response
	OnReceiveHeaders(headers map[string]string, endOfStream bool)

	// Called with a decoded data
	// endStream supplies whether this is the last data
	OnReceiveData(data IoBuffer, endOfStream bool)

	// Called with a decoded trailers frame, implicitly ends the stream.
	OnReceiveTrailers(trailers map[string]string)

	// Called with when exception occurs
	OnDecodeError(err error, headers map[string]string)
}

// A connection runs multiple streams
type StreamConnection interface {
	// Dispatch incoming data
	// On data read scenario, it connects connection and stream by dispatching read buffer to stream, stream uses protocol decode data, and popup event to controller
	Dispatch(buffer IoBuffer)

	// Protocol on the connection
	Protocol() Protocol

	// Send go away to remote for graceful shutdown
	GoAway()
}

// A server side stream connection.
type ServerStreamConnection interface {
	StreamConnection
}

// A client side stream connection.
type ClientStreamConnection interface {
	StreamConnection

	// Create a new outgoing request stream
	// responseDecoder supplies the decoder listeners on decode event
	// StreamSender supplies the encoder to write the request
	NewStream(streamId string, responseDecoder StreamReceiver) StreamSender
}

// Stream connection event listener
type StreamConnectionEventListener interface {
	// Called on remote sends 'go away'
	OnGoAway()
}

// Stream connection event listener for server connection
type ServerStreamConnectionEventListener interface {
	StreamConnectionEventListener

	// return request stream decoder
	NewStream(streamId string, responseEncoder StreamSender) StreamReceiver
}

type StreamFilterBase interface {
	OnDestroy()
}

// Called by stream filter to interact with underlying stream
type StreamFilterCallbacks interface {
	// the originating connection
	Connection() Connection

	// Reset the underlying stream
	ResetStream()

	// Route for current stream
	Route() Route

	// Get stream id
	StreamId() string

	// Request info related to the stream
	RequestInfo() RequestInfo
}

// Stream encoder filter
type StreamSenderFilter interface {
	StreamFilterBase

	// Encode headers
	// endStream supplies whether this is a header only request/response
	AppendHeaders(headers interface{}, endStream bool) FilterHeadersStatus

	// Called with data to be encoded
	// endStream supplies whether this is the last data
	AppendData(buf IoBuffer, endStream bool) FilterDataStatus

	// Called with trailers to be encoded, implicitly ending the stream
	AppendTrailers(trailers map[string]string) FilterTrailersStatus

	// Set StreamSenderFilterCallbacks
	SetEncoderFilterCallbacks(cb StreamSenderFilterCallbacks)
}

type StreamSenderFilterCallbacks interface {
	StreamFilterCallbacks

	// Continue iterating through the filter chain with buffered headers and body data
	ContinueEncoding()

	// data buffered by this filter or previous ones in the filter chain
	EncodingBuffer() IoBuffer

	// Add buffered body data
	AddEncodedData(buf IoBuffer, streamingFilter bool)

	// Set the buffer limit
	SetEncoderBufferLimit(limit uint32)

	// Get buffer limit
	EncoderBufferLimit() uint32
}

// Stream decoder filter
type StreamReceiverFilter interface {
	StreamFilterBase

	// Called with decoded headers
	// endStream supplies whether this is a header only request/response
	OnDecodeHeaders(headers map[string]string, endStream bool) FilterHeadersStatus

	// Called with a decoded data
	// endStream supplies whether this is the last data
	OnDecodeData(buf IoBuffer, endStream bool) FilterDataStatus

	// Called with decoded trailers, implicitly ending the stream
	OnDecodeTrailers(trailers map[string]string) FilterTrailersStatus

	// Set decoder filter callbacks
	SetDecoderFilterCallbacks(cb StreamReceiverFilterCallbacks)
}

// Stream decoder filter callbacks add additional callbacks that allow a decoding filter to restart
// decoding if they decide to hold data
type StreamReceiverFilterCallbacks interface {
	StreamFilterCallbacks

	// Continue iterating through the filter chain with buffered headers and body data
	// It can only be called if decode process has been stopped by current filter, using StopIteration from decodeHeaders() or StopIterationAndBuffer or StopIterationNoBuffer from decodeData()
	// The controller will dispatch headers and any buffered body data to the next filter in the chain.
	ContinueDecoding()

	// data buffered by this filter or previous ones in the filter chain
	// Nil if nothing has been buffered
	DecodingBuffer() IoBuffer

	// Add buffered body data
	AddDecodedData(buf IoBuffer, streamingFilter bool)

	// Called with headers to be encoded, optionally indicating end of stream
	// Filter uses this function to send out request/response headers of the stream
	// endStream supplies whether this is a header only request/response
	AppendHeaders(headers interface{}, endStream bool)

	// Called with data to be encoded, optionally indicating end of stream.
	// Filter uses this function to send out request/response data of the stream
	// endStream supplies whether this is the last data
	AppendData(buf IoBuffer, endStream bool)

	// Called with trailers to be encoded, implicitly ends the stream.
	// Filter uses this function to send out request/response trailers of the stream
	AppendTrailers(trailers map[string]string)

	// Set the buffer limit for decoder filters
	SetDecoderBufferLimit(limit uint32)

	// Get decoder buffer limit
	DecoderBufferLimit() uint32
}

type StreamFilterChainFactory interface {
	CreateFilterChain(context context.Context, callbacks FilterChainFactoryCallbacks)
}

type FilterChainFactoryCallbacks interface {
	AddStreamSenderFilter(filter StreamSenderFilter)

	AddStreamReceiverFilter(filter StreamReceiverFilter)
}

type FilterHeadersStatus string

const (
	// Continue filter chain iteration.
	FilterHeadersStatusContinue FilterHeadersStatus = "Continue"
	// Do not iterate to next iterator. Filter calls continueDecoding to continue.
	FilterHeadersStatusStopIteration FilterHeadersStatus = "StopIteration"
)

type FilterDataStatus string

const (
	// Continue filter chain iteration
	FilterDataStatusContinue FilterDataStatus = "Continue"
	// Do not iterate to next iterator, and buffer body data in controller for later use
	FilterDataStatusStopIterationAndBuffer FilterDataStatus = "StopIterationAndBuffer"
	// Do not iterate to next iterator, and buffer body data in controller for later use
	FilterDataStatusStopIterationAndWatermark FilterDataStatus = "StopIterationAndWatermark"
	// Do not iterate to next iterator, but do not buffer any of the body data in controller for later use
	FilterDataStatusStopIterationNoBuffer FilterDataStatus = "StopIterationNoBuffer"
)

type FilterTrailersStatus string

const (
	// Continue filter chain iteration
	FilterTrailersStatusContinue FilterTrailersStatus = "Continue"
	// Do not iterate to next iterator. Filter calls continueDecoding to continue.
	FilterTrailersStatusStopIteration FilterTrailersStatus = "StopIteration"
)

type PoolFailureReason string

const (
	Overflow          PoolFailureReason = "Overflow"
	ConnectionFailure PoolFailureReason = "ConnectionFailure"
)

type ConnectionPool interface {
	Protocol() Protocol

	DrainConnections()

	NewStream(context context.Context, streamId string,
		responseDecoder StreamReceiver, cb PoolEventListener) Cancellable

	Close()
}

type PoolEventListener interface {
	OnFailure(streamId string, reason PoolFailureReason, host Host)

	OnReady(streamId string, requestEncoder StreamSender, host Host)
}

type Cancellable interface {
	Cancel()
}
