package router

import (
	"regexp"
	
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"github.com/markphelps/optional"
)

type VirtualHostImpl struct {
	virtualHostName       string
	routes                []RouteBase //route impl
	virtualClusters       []VirtualClusterEntry
	sslRequirements       types.SslRequirements
	corsPolicy            types.CorsPolicy
	globalRouteConfig     *ConfigImpl
	requestHeadersParser  *HeaderParser
	responseHeadersParser *HeaderParser
}

func (vh *VirtualHostImpl) Name() string {
	
	return vh.virtualHostName
}

func (vh *VirtualHostImpl) CorsPolicy() types.CorsPolicy {
	
	return nil
}

func (vh *VirtualHostImpl) RateLimitPolicy() types.RateLimitPolicy {
	
	return nil
}

func (vh *VirtualHostImpl) GetRouteFromEntries(headers map[string]string, randomValue uint64) types.Route {
	// todo check tls
	for _, route := range vh.routes {
		
		if routeEntry := route.Match(headers, randomValue); routeEntry != nil {
			return routeEntry
		}
	}
	
	return nil
}

type VirtualClusterEntry struct {
	pattern regexp.Regexp
	method  optional.String
	name    string
}

func (vce *VirtualClusterEntry) VirtualClusterName() string {
	
	return vce.name
}