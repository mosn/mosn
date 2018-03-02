package codec

import (
"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type Codec_imp struct {
	protocol string

	decoder types.Decoder
}

func (code_ *Codec_imp) Decode(magic []byte) string {

	if len(magic) > 1 {

		if magic[0] == 1 {
			code_.protocol = "BOLT_V1"
		} else if magic[0] == 2 {
			code_.protocol = "BOLT_V2"
		} else if magic[0] == 0x13 {
			code_.protocol = "BOLT_V2"
		}

	}

}
