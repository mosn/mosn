package types

import "context"

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
