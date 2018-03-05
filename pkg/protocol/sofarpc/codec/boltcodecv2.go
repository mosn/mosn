package codec

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type BoltEncoderV2 struct {
}

type BoltDecoderV2 struct {
}

func (encoder *BoltEncoderV2) Encode(value interface{}, data types.IoBuffer) {

}

func (decoder *BoltDecoderV2) Decode(ctx interface{}, data types.IoBuffer, out interface{}) {
	fmt.Println("bolt v2 decode:", data.Bytes())
}
