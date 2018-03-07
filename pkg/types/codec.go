package types

type Protocols interface {
	Encode(value interface{}, data IoBuffer)

	Decode(data IoBuffer, filter DecodeFilter)
}

type Encoder interface {
	Encode(value interface{}, data IoBuffer)
}

type Decoder interface {
	//TODO replace ctx type with defined connection request context
	//TODO replace out type with defined pipeline output container
	Decode(ctx interface{}, data IoBuffer, out interface{})
}

type DecodeFilter interface {
	OnDecodeHeader(headers map[string]string) FilterStatus

	OnDecodeData(data []byte) FilterStatus

	OnDecodeTrailer(trailers map[string]string) FilterStatus
}

type DecoderCallbacks interface {
	OnResponseValue(value interface{})
}

type DecoderFactory interface {
	Create(cb DecoderCallbacks) Decoder
}
