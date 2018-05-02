package types

import "context"

//
//   The bunch of interfaces are structure skeleton to build a high performance, extensible protocol stream architecture.
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
//   Core model in stream layer is stream, which manages process of a request and a corresponding response.
// 	 Event listeners can be installed into a stream to monitor event.
//	 Stream always has a encoder and decoder:
// 		- StreamEncoder encodes and sends out a request/response, flag 'endStream' means data is ready to sendout, no need to wait for further input.
//		- StreamDecoder receives and decodes a request/response. In other words, it's more like a decode listener to get called on data decoded.
//
//   Stream:
//   	- Event listener
// 			- StreamEventListener
//      - Encoder
// 			- StreamEncoder
// 		- Decoder
//			- StreamDecoder
//
//	 In order to meet the expansion requirements in the stream processing, StreamEncoderFilter and StreamDecoderFilter can be used to do a filter inject to encode/decode process.
//   Filter's method will be called on each stream process stage and returns a status(Continue/Stop) to effect the control flow.
//	 TODO: more comments on StreamEncoderFilter/StreamDecoderFilter
//
//   From an abstract perspective, stream represents a virtual process on underlying connection. To make stream interactive with connection, some intermediate object can be used.
//	 StreamConnection is the core model to connect connection system to stream system. As a example, when proxy reads binary data from connection, it dispatches data to StreamConnection to do protocol decode.
//   Specifically, ClientStreamConnection uses a NewStream to exchange StreamDecoder with StreamEncoder.
//   Engine provides a callbacks(StreamEncoderFilterCallbacks/StreamDecoderFilterCallbacks) to let filter interact with stream engine.
// 	 As a example, a encoder filter stopped the encode process, it can continue it by StreamEncoderFilterCallbacks.ContinueEncoding later. Actually, a filter engine is a encoder/decoder itself.
//
//   Below is the basic relation on stream and connection:
//    ----------------------------------------------------------------------------------------------------------
//   |																											|
//   | 	  EventListener       						EventListener												|
//   |        *|                                           |   *												|
//   |         |                                           |													|
//   |        1|        1    1  				 *     1   |   *												|
// 	 |	    Connection -------- StreamConnection ------- Stream													|
//   |        1|                   	 					   |1													|
//	 |         |                   						   |--------------------------------					|
//   |        *|                   						   |*           	 				|*					|
//   |	 ConnectionFilter    						StreamEncoder  						StreamDecoder			|
//   |													   |1				 				|1					|
// 	 |													   |				 				|					|
//	 |													   |*				 				|*					|
//	 |										 	 StreamDecoderFilter	   			StreamDecoderFilter			|
//	 |													   |1								|1					|
//	 |													   |								|					|
// 	 |													   |1								|1					|
//	 |										 StreamDecoderFilterCallbacks     	StreamDecoderFilterCallbacks	|
//   |																											|
//    ----------------------------------------------------------------------------------------------------------
//

type StreamResetReason string

const (
	StreamConnectionTermination StreamResetReason = "ConnectionTermination"
	StreamConnectionFailed      StreamResetReason = "ConnectionFailed"
	StreamLocalReset            StreamResetReason = "StreamLocalReset"
	StreamOverflow              StreamResetReason = "StreamOverflow"
	StreamRemoteReset           StreamResetReason = "StreamRemoteReset"
)

type StreamEventListener interface {
	OnResetStream(reason StreamResetReason)

	OnAboveWriteBufferHighWatermark()

	OnBelowWriteBufferLowWatermark()
}

type Stream interface {
	AddEventListener(streamEventListener StreamEventListener)

	RemoveEventListener(streamEventListener StreamEventListener)

	ResetStream(reason StreamResetReason)

	ReadDisable(disable bool)
}

type StreamProto string

type StreamEncoder interface {
	EncodeHeaders(headers interface{}, endStream bool) error

	EncodeData(data IoBuffer, endStream bool) error

	EncodeTrailers(trailers map[string]string) error

	GetStream() Stream
}

type StreamDecoder interface {
	OnDecodeHeaders(headers map[string]string, endStream bool)

	OnDecodeData(data IoBuffer, endStream bool)

	OnDecodeTrailers(trailers map[string]string)
}

type StreamConnection interface {
	Dispatch(buffer IoBuffer)

	Protocol() Protocol

	OnUnderlyingConnectionAboveWriteBufferHighWatermark()

	OnUnderlyingConnectionBelowWriteBufferLowWatermark()
}

type ServerStreamConnection interface {
	StreamConnection
}

type ClientStreamConnection interface {
	StreamConnection

	// return request stream encoder
	NewStream(streamId string, responseDecoder StreamDecoder) StreamEncoder
}

type StreamConnectionEventListener interface {
	OnGoAway()
}

type ServerStreamConnectionEventListener interface {
	StreamConnectionEventListener

	// return request stream decoder
	NewStream(streamId string, responseEncoder StreamEncoder) StreamDecoder
}

type StreamFilterBase interface {
	OnDestroy()
}

type StreamFilterCallbacks interface {
	Connection() Connection

	ResetStream()

	Route() Route

	StreamId() string

	RequestInfo() RequestInfo
}

type StreamEncoderFilter interface {
	StreamFilterBase

	EncodeHeaders(headers interface{}, endStream bool) FilterHeadersStatus

	EncodeData(buf IoBuffer, endStream bool) FilterDataStatus

	EncodeTrailers(trailers map[string]string) FilterTrailersStatus

	SetEncoderFilterCallbacks(cb StreamEncoderFilterCallbacks)
}

type StreamEncoderFilterCallbacks interface {
	StreamFilterCallbacks

	ContinueEncoding()

	EncodingBuffer() IoBuffer

	AddEncodedData(buf IoBuffer, streamingFilter bool)

	OnEncoderFilterAboveWriteBufferHighWatermark()

	OnEncoderFilterBelowWriteBufferLowWatermark()

	SetEncoderBufferLimit(limit uint32)

	EncoderBufferLimit() uint32
}

type StreamDecoderFilter interface {
	StreamFilterBase

	DecodeHeaders(headers map[string]string, endStream bool) FilterHeadersStatus

	DecodeData(buf IoBuffer, endStream bool) FilterDataStatus

	DecodeTrailers(trailers map[string]string) FilterTrailersStatus

	SetDecoderFilterCallbacks(cb StreamDecoderFilterCallbacks)
}

type StreamDecoderFilterCallbacks interface {
	StreamFilterCallbacks

	ContinueDecoding()

	DecodingBuffer() IoBuffer

	AddDecodedData(buf IoBuffer, streamingFilter bool)

	EncodeHeaders(headers interface{}, endStream bool)

	EncodeData(buf IoBuffer, endStream bool)

	EncodeTrailers(trailers map[string]string)

	OnDecoderFilterAboveWriteBufferHighWatermark()

	OnDecoderFilterBelowWriteBufferLowWatermark()

	AddDownstreamWatermarkCallbacks(cb DownstreamWatermarkEventListener)

	RemoveDownstreamWatermarkCallbacks(cb DownstreamWatermarkEventListener)

	SetDecoderBufferLimit(limit uint32)

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
	FilterHeadersStatusContinue      FilterHeadersStatus = "Continue"
	FilterHeadersStatusStopIteration FilterHeadersStatus = "StopIteration"
)

type FilterDataStatus string

const (
	FilterDataStatusContinue                  FilterDataStatus = "Continue"
	FilterDataStatusStopIterationAndBuffer    FilterDataStatus = "StopIterationAndBuffer"
	FilterDataStatusStopIterationAndWatermark FilterDataStatus = "StopIterationAndWatermark"
	FilterDataStatusStopIterationNoBuffer     FilterDataStatus = "StopIterationNoBuffer"
)

type FilterTrailersStatus string

const (
	FilterTrailersStatusContinue      FilterTrailersStatus = "Continue"
	FilterTrailersStatusStopIteration FilterTrailersStatus = "StopIteration"
)
