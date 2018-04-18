package types

type Protocols interface {
	// return 1. stream id if have one 2. headers bytes
	EncodeHeaders(headers interface{}) (string, IoBuffer)

	EncodeData(data IoBuffer) IoBuffer

	EncodeTrailers(trailers map[string]string) IoBuffer

	Decode(data IoBuffer, filter DecodeFilter)
}

type Encoder interface {
	// return 1. stream id if have one 2. headers bytes
	EncodeHeaders(headers interface{}) (string, IoBuffer)

	EncodeData(data IoBuffer) IoBuffer

	EncodeTrailers(trailers map[string]string) IoBuffer
}

type Decoder interface {
	// return 1. bytes decoded 2. decoded cmd
	Decode(data IoBuffer) (int, interface{})
}

type DecodeFilter interface {
	OnDecodeHeader(streamId string, headers map[string]string) FilterStatus

	OnDecodeData(streamId string, data IoBuffer) FilterStatus

	OnDecodeTrailer(streamId string, trailers map[string]string) FilterStatus
}

type DecoderCallbacks interface {
	OnResponseValue(value interface{})
}

type DecoderFactory interface {
	Create(cb DecoderCallbacks) Decoder
}
