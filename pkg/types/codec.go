package types

import "bytes"

type Encoder interface {
	Encode(value interface{}, data *bytes.Buffer)
}

type Decoder interface {
	//TODO replace ctx type with defined connection request context
	//TODO replace out type with defined pipeline output container
	Decode(ctx interface{}, data *bytes.Buffer, out interface{})
}

type DecoderCallbacks interface {
	OnResponseValue(value interface{})
}

type DecoderFactory interface {
	Create(cb DecoderCallbacks) Decoder
}
