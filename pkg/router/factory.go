package router

import (
	"fmt"
	"errors"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type configFactory func(config interface{}) (types.Routers, error)

var routerConfigFactories map[types.Protocol]configFactory

func RegisteRouterConfigFactory(port types.Protocol, factory configFactory) {
	if routerConfigFactories == nil {
		routerConfigFactories = make(map[types.Protocol]configFactory)
	}

	if _, ok := routerConfigFactories[port]; !ok {
		routerConfigFactories[port] = factory
	}
}

func CreateRouteConfig(port types.Protocol, config interface{}) (types.Routers, error) {
	if factory, ok := routerConfigFactories[port]; ok {
		return factory(config)  //call NewBasicRoute
	} else {
		return nil, errors.New(fmt.Sprintf("Unsupported protocol %s", port))
	}
}