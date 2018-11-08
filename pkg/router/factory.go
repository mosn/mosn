package router

import (
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	RegisterRouterRule(DefaultSofaRouterRuleFactory)
}

var defaultRouterRuleFactory RouterRuleFactory

func RegisterRouterRule(f RouterRuleFactory) {
	defaultRouterRuleFactory = f
}

func DefaultSofaRouterRuleFactory(base *RouteRuleImplBase, headers []v2.HeaderMatcher) RouteBase {
	for _, header := range headers {
		if header.Name == types.SofaRouteMatchKey {
			return &SofaRouteRuleImpl{
				RouteRuleImplBase: base,
				matchValue:        header.Value,
			}
		}
	}
	return nil
}
