package subprotocol

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
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
		log.DefaultLogger.Tracef("create sub protocol codec %v success", prot)
		return spc.CreateSubProtocolCodec(context)
	}
	log.DefaultLogger.Tracef("unknown sub protocol = %v", prot)
	return nil
}
