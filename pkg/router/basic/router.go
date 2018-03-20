package basic

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/router"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"errors"
	"regexp"
)

func init() {
	router.RegisteRouterConfigFactory(protocol.SofaRpc, NewBasicRouter)
	router.RegisteRouterConfigFactory(protocol.Http2, NewBasicRouter)
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

type basicRouter struct {
	RouteRuleImplAdaptor
	name    string
	service string
	cluster string
}

func NewBasicRouter(config interface{}) (types.RouterConfig, error) {
	if config, ok := config.(*v2.Proxy); ok {
		routers := make([]router.RouteBase, 0)

		for _, r := range config.Routes {
			routers = append(routers, &basicRouter{
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

func (srr *basicRouter) Match(headers map[string]string) types.Route {
	if headers == nil {
		return nil
	}

	var ok bool
	var service string

	if service, ok = headers["Service"]; !ok {
		if service, ok = headers["service"]; !ok {
			return nil
		}
	}

	if match, _ := regexp.MatchString(srr.service, service); match {
		return srr
	} else {
		return nil
	}
}

func (srr *basicRouter) RedirectRule() types.RedirectRule {
	return nil
}

func (srr *basicRouter) RouteRule() types.RouteRule {
	return srr
}

func (srr *basicRouter) TraceDecorator() types.TraceDecorator {
	return nil
}

func (srr *basicRouter) ClusterName() string {
	return srr.cluster
}
