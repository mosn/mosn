package codec

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// types.Encoder & types.Decoder
type trCodec struct {
}

func (encoder *trCodec) Encode(value interface{}, data types.IoBuffer) {}

func (decoder *trCodec) Decode(ctx interface{}, data types.IoBuffer, out interface{}) {
	fmt.Println("tr decode:", data.Bytes())
}
