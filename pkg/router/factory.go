package router

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"errors"
	"fmt"
)

type configFactory func(config interface{}) (types.RouterConfig, error)

var routerConfigFactories map[protocol.Protocol]configFactory

func RegisteRouterConfigFactory(port protocol.Protocol, factory configFactory) {
	if routerConfigFactories == nil {
		routerConfigFactories = make(map[protocol.Protocol]configFactory)
	}

	if _, ok := routerConfigFactories[port]; !ok {
		routerConfigFactories[port] = factory
	}
}

func CreateRouteConfig(port protocol.Protocol, config interface{}) (types.RouterConfig, error) {
	if factory, ok := routerConfigFactories[port]; ok {
		return factory(config)
	} else {
		return nil, errors.New(fmt.Sprintf("Unsupported protocol %s", port))
	}
}
