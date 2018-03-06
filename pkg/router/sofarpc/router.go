package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/router"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"errors"
	"regexp"
)

func init() {
	router.RegisteRouterConfigFactory(protocol.SofaRpc, NewSofaRpcRouter)
}

// types.RouterConfig
type routeConfig struct {
	routers []router.RouteBase
}

func (rc routeConfig) Route(headers map[string]string) types.Route {
	for _, r := range rc.routers {
		if rule := r.Match(headers); rule != nil {
			return rule
		}
	}

	return nil
}

type simpleRpcRouter struct {
	RouteRuleImplAdaptor
	name    string
	service string
	cluster string
}

func NewSofaRpcRouter(config interface{}) (types.RouterConfig, error) {
	if config, ok := config.(*v2.RpcProxy); ok {
		routers := make([]router.RouteBase, 0)

		for _, r := range config.Routes {
			routers = append(routers, &simpleRpcRouter{
				name:    r.Name,
				service: r.Service,
				cluster: r.Cluster,
			})
		}

		return &routeConfig{
			routers,
		}, nil
	} else {
		return nil, errors.New("invalid config struct")
	}
}

func (srr *simpleRpcRouter) Match(headers map[string]string) types.Route {
	if headers == nil {
		return nil
	}

	var ok bool
	var service string

	if service, ok = headers["service"]; !ok {
		return nil
	}

	if match, _ := regexp.MatchString(srr.service, service); match {
		return srr
	} else {
		return nil
	}
}

func (srr *simpleRpcRouter) RedirectRule() types.RedirectRule {
	return nil
}

func (srr *simpleRpcRouter) RouteRule() types.RouteRule {
	return srr
}

func (srr *simpleRpcRouter) TraceDecorator() types.TraceDecorator {
	return nil
}

func (srr *simpleRpcRouter) ClusterName() string {
	return srr.cluster
}