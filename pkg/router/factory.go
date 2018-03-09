package router

import (
	"fmt"
	"errors"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type configFactory func(config interface{}) (types.RouterConfig, error)

var routerConfigFactories map[types.Protocol]configFactory

func RegisteRouterConfigFactory(port types.Protocol, factory configFactory) {
	if routerConfigFactories == nil {
		routerConfigFactories = make(map[types.Protocol]configFactory)
	}

	if _, ok := routerConfigFactories[port]; !ok {
		routerConfigFactories[port] = factory
	}
}

func CreateRouteConfig(port types.Protocol, config interface{}) (types.RouterConfig, error) {
	if factory, ok := routerConfigFactories[port]; ok {
		return factory(config)
	} else {
		return nil, errors.New(fmt.Sprintf("Unsupported protocol %s", port))
	}
}
