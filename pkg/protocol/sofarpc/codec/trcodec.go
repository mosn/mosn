package codec

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type TrEncoder struct {
}

type TrDecoder struct {
}

func (encoder *TrEncoder) Encode(value interface{}, data types.IoBuffer) {

}

func (decoder *TrDecoder) Decode(ctx interface{}, data types.IoBuffer, out interface{}) {
	fmt.Println("tr decode:", data.Bytes())
}
