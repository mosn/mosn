package network

import (
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

func init() {
	ConnNewPoolFactories = make(map[types.ProtocolName]connNewPool)
}

type connNewPool func(host types.Host) types.ConnectionPool

var ConnNewPoolFactories map[types.ProtocolName]connNewPool

func RegisterNewPoolFactory(protocol types.ProtocolName, factory connNewPool) {
	//other
	log.DefaultLogger.Infof("[network] [ register pool factory] register protocol: %v factory", protocol)
	ConnNewPoolFactories[protocol] = factory
}
