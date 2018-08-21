package xprotocol

import (
	"github.com/alipay/sofa-mosn/pkg/types"
	"context"
)

var subProtocolFactories map[types.SubProtocol]SubProtocolCodecFactory

func init() {
	subProtocolFactories = make(map[types.SubProtocol]SubProtocolCodecFactory)
}

func Register(prot types.SubProtocol, factory SubProtocolCodecFactory) {
	subProtocolFactories[prot] = factory
}

func CreateSubProtocolCodec(context context.Context, prot types.SubProtocol) types.Multiplexing {

	if spc, ok := subProtocolFactories[prot]; ok {
		return spc.CreateSubProtocolCodec(context)
	}
	return nil
}
