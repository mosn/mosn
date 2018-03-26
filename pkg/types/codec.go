package types

type Protocols interface {
	//Encode(value interface{}, data IoBuffer)

	// return stream id if have one
	Encode(value interface{}, data IoBuffer) uint32

	Decode(data IoBuffer, filter DecodeFilter)
}

type Encoder interface {
	// return stream id if have one
	Encode(value interface{}, data IoBuffer) uint32
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
}

type DecoderCallbacks interface {
	OnResponseValue(value interface{})
}

type DecoderFactory interface {
	Create(cb DecoderCallbacks) Decoder
}
