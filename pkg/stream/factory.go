package stream

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

var streamFactories map[types.Protocol]ProtocolStreamFactory

func init() {
	streamFactories = make(map[types.Protocol]ProtocolStreamFactory)
}

func Register(prot types.Protocol, factory ProtocolStreamFactory) {
	streamFactories[prot] = factory
}

func CreateServerStreamConnection(prot types.Protocol, connection types.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {

	if ssc, ok := streamFactories[prot]; ok {
		return ssc.CreateServerStream(connection, callbacks)
	} else {
		return nil
	}
}
