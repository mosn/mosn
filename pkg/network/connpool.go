package network

import (
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

func init() {
	ConnNewPoolFactories = make(map[types.ProtocolName]poolCreator)
}

type poolCreator struct {
	protocol types.ProtocolName
	createFunc  poolCreateFunc
}

type poolCreateFunc func(protocol types.ProtocolName, subProto types.ProtocolName, host types.Host) types.ConnectionPool

var ConnNewPoolFactories map[types.ProtocolName]poolCreator

func RegisterNewPoolFactory(protocol types.ProtocolName, factory poolCreateFunc) {
	//other
	log.DefaultLogger.Infof("[network] [ register pool factory] register protocol: %v factory", protocol)
	ConnNewPoolFactories[protocol] = poolCreator{
		createFunc: factory,
		protocol : protocol,
	}
}

// CreatePool creates a pool for a host
func (c poolCreator) CreatePool(host types.Host, subProtocol types.ProtocolName) types.ConnectionPool{
	return c.createFunc(c.protocol, subProtocol, host)
}
