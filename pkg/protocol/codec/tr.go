package codec

import (
	"bytes"
	"fmt"
)

type TrEncoder struct {
}

type TrDecoder struct {
}

func (encoder *TrEncoder) Encode(value interface{}, data *bytes.Buffer) {

}

func (decoder *TrDecoder) Decode(ctx interface{}, data *bytes.Buffer, out interface{}) {
	fmt.Println("tr decode:", data.Bytes())
}
