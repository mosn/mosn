package types

import (
	"bytes"
	"bufio"
)

type StreamResetReason string

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

	EncodeData(data bytes.Buffer, endStream bool)

	EncodeTrailers(trailers map[string]string)

	GetStream() Stream
}

type StreamDecoder interface {
	DecodeHeaders(hbr *bufio.Reader, endStream bool)

	DecodeData(dr *bufio.Reader, endStream bool)

	DecodeTrailers(trailers map[string]string)
}

type StreamConnection interface {
	Dispatch(buffer *bytes.Buffer)

	GoAway()

	Protocol() string

	ShutdownNotice()

	WantsToWrite()

	OnUnderlyingConnectionAboveWriteBufferHighWatermark()

	OnUnderlyingConnectionBelowWriteBufferLowWatermark()
}

type ServerConnection interface {
	Connection
}

type StreamClientConnection interface {
	Connection

	newStream(responseDecoder StreamDecoder) StreamEncoder
}


type StreamFilterBase interface {
	onDestroy()
}

type StreamEncoderFilter interface {
	StreamFilterBase

	EncodeHeaders(headers map[string]string, endStream bool) FilterHeadersStatus

	EncodeData(dr *bufio.Reader, endStream bool) FilterDataStatus

	EncodeTrailers(trailers map[string]string) FilterTrailersStatus

	SetEncoderFilterCallbacks(cb StreamEncoderFilterCallbacks)
}

type StreamEncoderFilterCallbacks interface {
	ContinueEncoding()

	EncodingBuffer() *bufio.Reader

	AddEncodedData(buffer *bufio.Reader, streamingFilter bool)

	OnEncoderFilterAboveWriteBufferHighWatermark()

	OnEncoderFilterBelowWriteBufferLowWatermark()

	SetEncoderBufferLimit(limit uint32)

	EncoderBufferLimit() uint32
}

type StreamDecoderFilter interface {
	StreamFilterBase

	DecodeHeaders(headers map[string]string, endStream bool) FilterHeadersStatus

	DecodeData(dr *bufio.Reader, endStream bool) FilterDataStatus

	DecodeTrailers(trailers map[string]string) FilterTrailersStatus

	SetDecoderFilterCallbacks(cb StreamDecoderFilterCallbacks)
}

type StreamDecoderFilterCallbacks interface {
	ContinueDecoding()

	DecodingBuffer() *bufio.Reader

	AddDecodedData(data *bufio.Reader, streamingFilter bool)

	EncodeHeaders(dr *bufio.Reader, endStream bool)

	EncodeData(data bytes.Buffer, endStream bool)

	EncodeTrailers(trailers *bufio.Reader)

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
