package types

import "bytes"

type Encoder interface {
	Encode(value interface{}, data bytes.Buffer)
}

type Decoder interface {
	Decode(data bytes.Buffer)
}

type DecoderCallbacks interface {
	OnResponseValue(value interface{})
}

type DecoderFactory interface {
	Create(cb DecoderCallbacks) Decoder
}



