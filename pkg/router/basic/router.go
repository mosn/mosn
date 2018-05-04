package basic

import (
	"errors"
	"time"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/router"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
)

func init() {
	router.RegisteRouterConfigFactory(protocol.SofaRpc, NewBasicRouter)
	router.RegisteRouterConfigFactory(protocol.Http2, NewBasicRouter)
}

// types.RouterConfig
type routeConfig struct {
	routers []router.RouteBase

}

//update route when cluster changed
func (rc *routeConfig) OnClusterChanged(serviceName string, clusterName string) {
	log.DefaultLogger.Debugf("[Debuginfo]cluster changed servicename:%s clustername:%s ", serviceName, clusterName)
	for _, r := range rc.routers {

		if r.ClusterName() == clusterName {
			log.DefaultLogger.Debugf("[DEBUG INFO],cluster %s already exit", clusterName)
			//r.UpdateServiceName(serviceName)
			return
		} else {
			//create a new basic route
			routeName := rc.GetRouteNameByCluster(clusterName)
			router := &basicRouter{
				name:    routeName,
				service: serviceName,
				cluster: clusterName,
			}
			router.policy = &routerPolicy{
				retryOn:      false,
				retryTimeout: 0,
				numRetries:   0,
			}

			rc.routers = append(rc.routers, router)
		}
	}
}

//at this time, just use echo
func (srr *routeConfig) GetRouteNameByCluster(clusterName string) string {
	return clusterName
}

//Use for Routing, need completing
func (rc *routeConfig) Route(headers map[string]string) types.Route {
	log.DefaultLogger.Debugf("[DebugInfo]",rc.routers)
	for _, r := range rc.routers {
		if rule := r.Match(headers); rule != nil {
			return rule
		}
	}
	return nil
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

//Called by NewProxy
func NewBasicRouter(config interface{}) (types.RouterConfig, error) {
	if config, ok := config.(*v2.Proxy); ok {
		routers := make([]router.RouteBase, 0)

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

		rc := &routeConfig {
			routers,
		}
		//注册cluster变动时候的监听器
		cluster.ClusterAdap.RegisterClusterChangeListener(rc)
		return rc, nil
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

	if srr.service == service {
		return srr
	} else {
		return nil
	}

	return srr
}

//update cluster's service name if needed
func (srr *basicRouter) UpdateServiceName(serviceName string) {
	srr.service = serviceName
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

func (srr *basicRouter) GlobalTimeout() time.Duration {
	return srr.globalTimeout
}

func (srr *basicRouter) Policy() types.Policy {
	return srr.policy
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
