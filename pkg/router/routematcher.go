package router

import (
	"strings"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func init() {
	RegisteRouterConfigFactory(protocol.SofaRpc, NewRouteMatcher)
	RegisteRouterConfigFactory(protocol.Http2, NewRouteMatcher)
}

func NewRouteMatcher(config interface{}) (types.Routers, error) {
	routerMatcher := &RouteMatcher{
		virtualHosts: make(map[string]types.VirtualHost),
	}

	if config, ok := config.(*v2.Proxy); ok {

		for _, virtualHost := range config.VirtualHosts {

			//todo 补充virtual host 其他成员
			vh := NewVirtualHostImpl(virtualHost, config.ValidateClusters)

			for _, domain := range virtualHost.Domains {
				if domain == "*" {
					if routerMatcher.defaultVirtualHost != nil {
						log.DefaultLogger.Errorf("Only a single wildcard domain permitted")
					}

					routerMatcher.defaultVirtualHost = vh
				} else if _, ok := routerMatcher.virtualHosts[domain]; ok {
					log.DefaultLogger.Errorf("Only unique values for domains are permitted, get duplicate domain = %s",
						domain)
				} else {
					routerMatcher.virtualHosts[domain] = vh
				}
			}
		}
	}

	return routerMatcher, nil
}

// A router wrapper used to matches an incoming request headers to a backend cluster
type RouteMatcher struct {
	virtualHosts                map[string]types.VirtualHost // key: host
	defaultVirtualHost          types.VirtualHost
	wildcardVirtualHostSuffixes map[int64]map[string]types.VirtualHost
}

// Routing with Virtual Host
func (rm *RouteMatcher) Route(headers map[string]string, randomValue uint64) types.Route {
	// First Step: Select VirtualHost with "host" in Headers form VirtualHost Array
	virtualHost := rm.findVirtualHost(headers)

	if virtualHost == nil {
		log.DefaultLogger.Warnf("No VirtualHost Found when Routing, But Use Default Virtual Host, Request Headers = %+v", headers)
	}

	// Second Step: Match Route from Routes in a Virtual Host
	return virtualHost.GetRouteFromEntries(headers, randomValue)
}

func (rm *RouteMatcher) findVirtualHost(headers map[string]string) types.VirtualHost {
	if len(rm.virtualHosts) == 0 && rm.defaultVirtualHost != nil {

		return rm.defaultVirtualHost
	}

	host := strings.ToLower(headers[protocol.MosnHeaderHostKey])

	// for service, header["host"] == header["service"] == servicename
	// or use only a unique key for sofa's virtual host
	if virtualHost, ok := rm.virtualHosts[host]; ok {
		return virtualHost
	}

	// todo support WildcardVirtualHost

	return rm.defaultVirtualHost
}

// todo match wildcard
func (rm *RouteMatcher) findWildcardVirtualHost(host string) types.VirtualHost {

	return nil
}

func (rm *RouteMatcher) AddRouter(routerName string) {}

func (rm *RouteMatcher) DelRouter(routerName string) {}
