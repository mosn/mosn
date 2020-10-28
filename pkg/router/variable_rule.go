package router

import (
	"context"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/variable"
)

type VariableRouteRuleImpl struct {
	*RouteRuleImplBase
	VariableMap map[string]string
}

func (vrri *VariableRouteRuleImpl) PathMatchCriterion() api.PathMatchCriterion {
	return vrri
}

func (vrri *VariableRouteRuleImpl) RouteRule() api.RouteRule {
	return vrri
}

func (vrri *VariableRouteRuleImpl) Matcher() string {
	return ""
}

func (vrri *VariableRouteRuleImpl) MatchType() api.PathMatchType {
	return api.Variable
}

func (vrri *VariableRouteRuleImpl) FinalizeRequestHeaders(headers api.HeaderMap, requestInfo api.RequestInfo) {
}

func (vrri *VariableRouteRuleImpl) Match(ctx context.Context, headers api.HeaderMap, randomValue uint64) api.Route {

	var err error
	var value string
	for k, v := range vrri.VariableMap {
		value, err = variable.GetVariableValue(ctx, k)
		if err != nil {
			continue
		}
		if value == v{
			return vrri
		}
	}
	log.DefaultLogger.Errorf(RouterLogFormat, "variable rotue rule", "failed match", vrri.VariableMap)
	return nil

}
