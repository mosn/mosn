package codec

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// types.Encoder & types.Decoder
type boltV2Codec struct {
}

func (encoder *boltV2Codec) Encode(value interface{}, data types.IoBuffer) {
}

func (decoder *boltV2Codec) Decode(ctx interface{}, data types.IoBuffer, out interface{}) {
	fmt.Println("bolt v2 decode:", data.Bytes())
}
