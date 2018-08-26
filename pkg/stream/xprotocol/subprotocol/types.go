package subprotocol

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type SubProtocolCodecFactory interface {
	CreateSubProtocolCodec(context context.Context) types.Multiplexing
}
