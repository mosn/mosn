package subprotocol

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/types"
)

// CodecFactory subprotocol plugin factory
type CodecFactory interface {
	CreateSubProtocolCodec(context context.Context) types.Multiplexing
}
