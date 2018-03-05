package codec

import (
	"bytes"
	"fmt"
)

type BoltEncoderV2 struct {
}

type BoltDecoderV2 struct {
}

func (encoder *BoltEncoderV2) Encode(value interface{}, data *bytes.Buffer) {

}

func (decoder *BoltDecoderV2) Decode(ctx interface{}, data *bytes.Buffer, out interface{}) {
	fmt.Println("bolt v2 decode:", data.Bytes())
}
