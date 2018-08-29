package subprotocol

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var subProtocolFactories map[types.SubProtocol]CodecFactory

func init() {
	subProtocolFactories = make(map[types.SubProtocol]CodecFactory)
}

// Register SubProtocol Plugin
func Register(prot types.SubProtocol, factory CodecFactory) {
	subProtocolFactories[prot] = factory
}

// CreateSubProtocolCodec return SubProtocol Codec
func CreateSubProtocolCodec(context context.Context, prot types.SubProtocol) types.Multiplexing {

	if spc, ok := subProtocolFactories[prot]; ok {
		log.DefaultLogger.Tracef("create sub protocol codec %v success", prot)
		return spc.CreateSubProtocolCodec(context)
	}
	log.DefaultLogger.Errorf("unknown sub protocol = %v", prot)
	return nil
}
