package network

import "github.com/alipay/sofa-mosn/pkg/types"

func init() {
	ConnNewPoolFactories = make(map[types.Protocol]connNewPool)
}

type connNewPool func(host types.Host) types.ConnectionPool

var ConnNewPoolFactories map[types.Protocol]connNewPool

func RegisterNewPoolFactory(protocol types.Protocol, factory connNewPool) {
	//other
	ConnNewPoolFactories[protocol] = factory
}
