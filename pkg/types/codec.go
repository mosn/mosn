package types

type Encoder interface {
	Encode(value interface{}, data IoBuffer)
}

type Decoder interface {
	//TODO replace ctx type with defined connection request context
	//TODO replace out type with defined pipeline output container
	Decode(ctx interface{}, data IoBuffer, out interface{})
}

type DecoderCallbacks interface {
	OnResponseValue(value interface{})
}

type DecoderFactory interface {
	Create(cb DecoderCallbacks) Decoder
}
