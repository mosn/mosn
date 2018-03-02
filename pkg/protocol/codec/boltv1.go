package codec

import (
	"bytes"
	"fmt"
)

type BoltEncoderV1 struct {
}

type BoltDecoderV1 struct {
}

func (encoder *BoltEncoderV1) Encode(value interface{}, data *bytes.Buffer) {

}

func (decoder *BoltDecoderV1) Decode(ctx interface{}, data *bytes.Buffer, out interface{}) {
	fmt.Println("bolt v1 decode:", data.Bytes())
}
