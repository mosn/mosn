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
// 		- StreamEncoder: a sender encodes request/response to binary and sends it out, flag 'endStream' means data is ready to sendout, no need to wait for further input.
//		- StreamDecoder: It's more like a decode listener to get called on a receiver receives binary and decodes to a request/response.
//	 	- Stream does not have a predetermined direction, so StreamEncoder could be a request encoder as a client or a response encoder as a server. It's just about the scenario, so does StreamDecoder.
//
//   Stream:
//   	- Event listener
// 			- StreamEventListener
//      - Encoder
// 			- StreamEncoder
// 		- Decoder
//			- StreamDecoder
//
//	 In order to meet the expansion requirements in the stream processing, StreamEncoderFilters and StreamDecoderFilters are introduced as a filter chain in encode/decode process.
//   Filter's method will be called on corresponding stream process stage and returns a status(Continue/Stop) to effect the control flow.
//
//   From an abstract perspective, stream represents a virtual process on underlying connection. To make stream interactive with connection, some intermediate object can be used.
//	 StreamConnection is the core model to connect connection system to stream system. As a example, when proxy reads binary data from connection, it dispatches data to StreamConnection to do protocol decode.
//   Specifically, ClientStreamConnection uses a NewStream to exchange StreamDecoder with StreamEncoder.
//   Engine provides a callbacks(StreamEncoderFilterCallbacks/StreamDecoderFilterCallbacks) to let filter interact with stream engine.
// 	 As a example, a encoder filter stopped the encode process, it can continue it by StreamEncoderFilterCallbacks.ContinueEncoding later. Actually, a filter engine is a encoder/decoder itself.
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
//   |	 ConnectionFilter    		   |			 StreamEncoder[sender]  		StreamDecoder[receiver]			|
//   |								   |*					   |1				 				|1					|
// 	 |						StreamConnectionEventListener	   |				 				|					|
//	 |													       |*				 				|*					|
//	 |										 	 		StreamDecoderFilter	   			StreamDecoderFilter			|
//	 |													   	   |1								|1					|
//	 |													   	   |								|					|
// 	 |													       |1								|1					|
//	 |										 		StreamDecoderFilterCallbacks     StreamDecoderFilterCallbacks	|
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

	// Called when a stream , or the connection the stream is sending to, goes over its high watermark.
	OnAboveWriteBufferHighWatermark()

	// Called when a stream, or the connection the stream is sending to, goes from over its high watermark to under its low watermark
	OnBelowWriteBufferLowWatermark()
}

// StreamEncoder encodes protocol stream
// On server scenario, StreamEncoder handles response
// On client scenario, StreamEncoder handles request
type StreamEncoder interface {
	// Encode headers
	// endStream supplies whether this is a header only request/response
	EncodeHeaders(headers interface{}, endStream bool) error

	// Encode data
	// endStream supplies whether this is the last data frame
	EncodeData(data IoBuffer, endStream bool) error

	// Encode trailers, implicitly ends the stream.
	EncodeTrailers(trailers map[string]string) error

	// Get related stream
	GetStream() Stream
}

// Listeners called on decode stream event
// On server scenario, StreamDecoder handles request
// On client scenario, StreamEncoder handles response
type StreamDecoder interface {
	// Called with decoded headers
	// endStream supplies whether this is a header only request/response
	OnDecodeHeaders(headers map[string]string, endStream bool)

	// Called with a decoded data
	// endStream supplies whether this is the last data
	OnDecodeData(data IoBuffer, endStream bool)

	// Called with a decoded trailers frame, implicitly ends the stream.
	OnDecodeTrailers(trailers map[string]string)

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

	// Called when the underlying Connection goes over its high watermark.
	OnUnderlyingConnectionAboveWriteBufferHighWatermark()

	// Called when the underlying Connection goes from over its high watermark to under its low watermark.
	OnUnderlyingConnectionBelowWriteBufferLowWatermark()
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
	// StreamEncoder supplies the encoder to write the request
	NewStream(streamId string, responseDecoder StreamDecoder) StreamEncoder
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
	NewStream(streamId string, responseEncoder StreamEncoder) StreamDecoder
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
type StreamEncoderFilter interface {
	StreamFilterBase

	// Encode headers
	// endStream supplies whether this is a header only request/response
	EncodeHeaders(headers interface{}, endStream bool) FilterHeadersStatus

	// Called with data to be encoded
	// endStream supplies whether this is the last data
	EncodeData(buf IoBuffer, endStream bool) FilterDataStatus

	// Called with trailers to be encoded, implicitly ending the stream
	EncodeTrailers(trailers map[string]string) FilterTrailersStatus

	// Set StreamEncoderFilterCallbacks
	SetEncoderFilterCallbacks(cb StreamEncoderFilterCallbacks)
}

type StreamEncoderFilterCallbacks interface {
	StreamFilterCallbacks

	// Continue iterating through the filter chain with buffered headers and body data
	ContinueEncoding()

	// data buffered by this filter or previous ones in the filter chain
	EncodingBuffer() IoBuffer

	// Add buffered body data
	AddEncodedData(buf IoBuffer, streamingFilter bool)

	// Called when an encoder filter goes over its high watermark
	OnEncoderFilterAboveWriteBufferHighWatermark()

	// Called when a encoder filter goes from over its high watermark to under its low watermark
	OnEncoderFilterBelowWriteBufferLowWatermark()

	// Set the buffer limit
	SetEncoderBufferLimit(limit uint32)

	// Get buffer limit
	EncoderBufferLimit() uint32
}

// Stream decoder filter
type StreamDecoderFilter interface {
	StreamFilterBase

	// Called with decoded headers
	// endStream supplies whether this is a header only request/response
	DecodeHeaders(headers map[string]string, endStream bool) FilterHeadersStatus

	// Called with a decoded data
	// endStream supplies whether this is the last data
	DecodeData(buf IoBuffer, endStream bool) FilterDataStatus

	// Called with decoded trailers, implicitly ending the stream
	DecodeTrailers(trailers map[string]string) FilterTrailersStatus

	// Set decoder filter callbacks
	SetDecoderFilterCallbacks(cb StreamDecoderFilterCallbacks)
}

// Stream decoder filter callbacks add additional callbacks that allow a decoding filter to restart
// decoding if they decide to hold data
type StreamDecoderFilterCallbacks interface {
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
	EncodeHeaders(headers interface{}, endStream bool)

	// Called with data to be encoded, optionally indicating end of stream.
	// Filter uses this function to send out request/response data of the stream
	// endStream supplies whether this is the last data
	EncodeData(buf IoBuffer, endStream bool)

	// Called with trailers to be encoded, implicitly ends the stream.
	// Filter uses this function to send out request/response trailers of the stream
	EncodeTrailers(trailers map[string]string)

	// Called when the buffer for a decoder filter or any buffers the filter sends data to go over their high watermark
	OnDecoderFilterAboveWriteBufferHighWatermark()

	// Called when a decoder filter or any buffers the filter sends data to go from over its high watermark to under its low watermark
	OnDecoderFilterBelowWriteBufferLowWatermark()

	// Called by a filter to subscribe to watermark events on the downstream stream and downstream connection
	AddDownstreamWatermarkCallbacks(cb DownstreamWatermarkEventListener)

	// Called by a filter to stop subscribing to watermark events on the downstream stream and downstream connection
	RemoveDownstreamWatermarkCallbacks(cb DownstreamWatermarkEventListener)

	// Set the buffer limit for decoder filters
	SetDecoderBufferLimit(limit uint32)

	// Get decoder buffer limit
	DecoderBufferLimit() uint32
}

type DownstreamWatermarkEventListener interface {
	OnAboveWriteBufferHighWatermark()

	OnBelowWriteBufferLowWatermark()
}

type StreamFilterChainFactory interface {
	CreateFilterChain(context context.Context, callbacks FilterChainFactoryCallbacks)
}

type FilterChainFactoryCallbacks interface {
	AddStreamDecoderFilter(filter StreamDecoderFilter)

	AddStreamEncoderFilter(filter StreamEncoderFilter)
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
		responseDecoder StreamDecoder, cb PoolEventListener) Cancellable

	Close()
	
	// Used to init active client for conn pool
	InitActiveClient(context context.Context) error
	// Used to get host of the conn pool
	Host() Host
}

type PoolEventListener interface {
	OnPoolFailure(streamId string, reason PoolFailureReason, host Host)

	OnPoolReady(streamId string, requestEncoder StreamEncoder, host Host)
}

type Cancellable interface {
	Cancel()
}
