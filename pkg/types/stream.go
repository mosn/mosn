package types

type StreamResetReason string

const (
	StreamConnectionTermination    StreamResetReason = "ConnectionTermination"
	StreamConnectionFailed         StreamResetReason = "ConnectionFailed"
	StreamLocalReset               StreamResetReason = "StreamLocalReset"
	StreamLocalRefusedStreamReset  StreamResetReason = "StreamLocalRefusedStreamReset"
	StreamOverflow                 StreamResetReason = "StreamOverflow"
	StreamRemoteReset              StreamResetReason = "StreamRemoteReset"
	StreanRemoteRefusedStreamReset StreamResetReason = "StreanRemoteRefusedStreamReset"
)

type StreamCallbacks interface {
	OnResetStream(reason StreamResetReason)

	OnAboveWriteBufferHighWatermark()

	OnBelowWriteBufferLowWatermark()
}

type Stream interface {
	AddCallbacks(streamCb StreamCallbacks)

	RemoveCallbacks(streamCb StreamCallbacks)

	ResetStream(reason StreamResetReason)

	ReadDisable(disable bool)

	BufferLimit() uint32
}

type StreamProto string

type StreamRecognizer interface {
	MagicBytesLength() int

	SnifferProto([]byte) StreamProto
}

type StreamEncoder interface {
	EncodeHeaders(headers map[string]string, endStream bool)

	EncodeData(data IoBuffer, endStream bool)

	EncodeTrailers(trailers map[string]string)

	GetStream() Stream
}

type StreamDecoder interface {
	DecodeHeaders(headers map[string]string, endStream bool)

	DecodeData(data IoBuffer, endStream bool)

	DecodeTrailers(trailers map[string]string)

	DecodeComplete(data IoBuffer)
}

type StreamConnection interface {
	Dispatch(buffer IoBuffer)

	//GoAway()

	Protocol() Protocol

	//ShutdownNotice()
	//
	//WantsToWrite()
	//
	//OnUnderlyingConnectionAboveWriteBufferHighWatermark()
	//
	//OnUnderlyingConnectionBelowWriteBufferLowWatermark()
}

type ServerStreamConnection interface {
	StreamConnection
}

type ClientStreamConnection interface {
	StreamConnection

	NewStream(streamId uint32, responseDecoder StreamDecoder) StreamEncoder
}

type StreamConnectionCallbacks interface {
	OnGoAway()
}

type ServerStreamConnectionCallbacks interface {
	NewStream(streamId uint32, responseEncoder StreamEncoder) StreamDecoder
}

type StreamFilterBase interface {
	onDestroy()
}

type StreamFilterCallbacks interface {
	Connection() Connection

	ResetStream()

	Route() Route

	ClearRouteCache()

	StreamId() string

	RequestInfo() RequestInfo
}

type StreamEncoderFilter interface {
	StreamFilterBase

	EncodeHeaders(headers map[string]string, endStream bool) FilterHeadersStatus

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

	EncodeHeaders(buf IoBuffer, endStream bool)

	EncodeData(buf IoBuffer, endStream bool)

	EncodeTrailers(trailers IoBuffer)

	OnDecoderFilterAboveWriteBufferHighWatermark()

	OnDecoderFilterBelowWriteBufferLowWatermark()

	AddDownstreamWatermarkCallbacks(cb DownstreamWatermarkCallbacks)

	RemoveDownstreamWatermarkCallbacks(cb DownstreamWatermarkCallbacks)

	SetDecoderBufferLimit(limit uint32)

	DecoderBufferLimit() uint32
}

type DownstreamWatermarkCallbacks interface {
	OnAboveWriteBufferHighWatermark()

	OnBelowWriteBufferLowWatermark()
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
