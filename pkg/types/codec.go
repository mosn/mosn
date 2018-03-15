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
	Decode(ctx interface{}, data IoBuffer, out interface{}) int
}

type DecodeFilter interface {
	OnDecodeHeader(streamId uint32, headers map[string]string) FilterStatus

	OnDecodeData(streamId uint32, data IoBuffer) FilterStatus

	OnDecodeTrailer(streamId uint32, trailers map[string]string) FilterStatus

	// just for raw request forward
	OnDecodeComplete(streamId uint32, buf IoBuffer)
}

type DecoderCallbacks interface {
	OnResponseValue(value interface{})
}

type DecoderFactory interface {
	Create(cb DecoderCallbacks) Decoder
}
