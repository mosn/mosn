package basic

import (
	"errors"
	"time"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/router"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func init() {
	router.RegisteRouterConfigFactory(protocol.SofaRpc, NewRouterConfig)
	router.RegisteRouterConfigFactory(protocol.Http2, NewRouterConfig)
}

// types.RouterConfig
type routeConfig struct {
	routers        []router.RouteBase
	supportDynamic bool
	dynamicRouter  DynamicRouter
}

//Routing, use static router first， then use dynamic router if support
func (rc *routeConfig) Route(headers map[string]string) (types.Route, string) {
	log.DefaultLogger.Debugf("[DebugInfo]", rc.routers)

	//use static route
	for _, r := range rc.routers {
		if rule, key := r.Match(headers); rule != nil {
			return rule, key
		}
	}

	//use dynamic route
	if rc.supportDynamic {
		return rc.dynamicRouter.Match(headers)
	}

	return nil, ""
}

// types.Route
// types.RouteRule
// router.Matchable
type basicRouter struct {
	RouteRuleImplAdaptor
	name          string
	service       string
	cluster       string
	globalTimeout time.Duration
	policy        *routerPolicy
}

type DynamicRouter struct {
	RouteRuleImplAdaptor
	globalTimeout time.Duration
	policy        *routerPolicy
}

//Called by NewProxy
func NewRouterConfig(config interface{}) (types.RouterConfig, error) {
	if config, ok := config.(*v2.Proxy); ok {
		routers := make([]router.RouteBase, 0)

		//new basic router
		for _, r := range config.Routes {
			router := &basicRouter{
				name:          r.Name,
				service:       r.Service,
				cluster:       r.Cluster,
				globalTimeout: r.GlobalTimeout,
			}

			if r.RetryPolicy != nil {
				router.policy = &routerPolicy{
					retryOn:      r.RetryPolicy.RetryOn,
					retryTimeout: r.RetryPolicy.RetryTimeout,
					numRetries:   r.RetryPolicy.NumRetries,
				}
			} else {
				// default
				router.policy = &routerPolicy{
					retryOn:      false,
					retryTimeout: 0,
					numRetries:   0,
				}
			}

			routers = append(routers, router)
		}

		//new dynamic route
		var dynamicRouter DynamicRouter
		if config.SupportDynamicRoute {
			dynamicRouter = DynamicRouter{
				//use static route's config for GlobalTimeout and Policy
				globalTimeout: routers[0].GlobalTimeout(),
				policy:        routers[0].Policy().(*routerPolicy),
			}
		}

		//new routeConfig
		rc := &routeConfig{
			routers:        routers,
			supportDynamic: config.SupportDynamicRoute,
			dynamicRouter:  dynamicRouter,
		}

		return rc, nil
	} else {
		return nil, errors.New("invalid config struct")
	}
}

func (srr *basicRouter) Match(headers map[string]string) (types.Route, string) {
	if headers == nil {
		return nil, ""
	}

	var ok bool
	var service string

	if service, ok = headers["Service"]; !ok {
		if service, ok = headers["service"]; !ok {
			return nil, ""
		}
	}

	if srr.service == service {
		return srr, service
	} else {
		return nil, service
	}
	return srr, service
}

func (drr *DynamicRouter) Match(headers map[string]string) (types.Route, string) {
	if headers == nil {
		return nil, ""
	}

	var ok bool
	var service string

	if service, ok = headers["Service"]; !ok {
		if service, ok = headers["service"]; !ok {
			return nil, ""
		}
	}
	return drr, service
}

func (srr *basicRouter) RedirectRule() types.RedirectRule {
	return nil
}

func (drr *DynamicRouter) RedirectRule() types.RedirectRule {
	return nil
}

func (srr *basicRouter) RouteRule() types.RouteRule {
	return srr
}

func (drr *DynamicRouter) RouteRule() types.RouteRule {
	return drr
}

func (srr *basicRouter) TraceDecorator() types.TraceDecorator {
	return nil
}

func (drr *DynamicRouter) TraceDecorator() types.TraceDecorator {
	return nil
}

func (srr *basicRouter) ClusterName(serviceName string) string {
	return srr.cluster
}

//use servicename
func (drr *DynamicRouter) ClusterName(serviceName string) string {
	return drr.MapRule(serviceName)
}

func (srr *basicRouter) GlobalTimeout() time.Duration {
	return srr.globalTimeout
}

func (drr *DynamicRouter) GlobalTimeout() time.Duration {
	return drr.globalTimeout
}

func (srr *basicRouter) Policy() types.Policy {
	return srr.policy
}

func (drr *DynamicRouter) Policy() types.Policy {
	return drr.policy
}

//For dynamic use echo at this time
func (drr *DynamicRouter)MapRule(routeKey string) string{
	return routeKey
}

type routerPolicy struct {
	retryOn      bool
	retryTimeout time.Duration
	numRetries   int
}

func (p *routerPolicy) RetryOn() bool {
	return p.retryOn
}

func (p *routerPolicy) TryTimeout() time.Duration {
	return p.retryTimeout
}

func (p *routerPolicy) NumRetries() int {
	return p.numRetries
}

func (p *routerPolicy) RetryPolicy() types.RetryPolicy {
	return p
}

func (p *routerPolicy) ShadowPolicy() types.ShadowPolicy {
	return nil
}

func (p *routerPolicy) CorsPolicy() types.CorsPolicy {
	return nil
}

func (p *routerPolicy) LoadBalancerPolicy() types.LoadBalancerPolicy {
	return nil
}
