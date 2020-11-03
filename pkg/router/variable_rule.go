package router

import (
	"context"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/variable"
	"regexp"
	"strings"
)

type Model string

const (
	AND Model = "and"
	OR  Model = "or"
)

type VariableRouteRuleImpl struct {
	*RouteRuleImplBase
	Variables []*VariableMatchItem
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
	result := true
	walkVarName := ""
	for _, v := range vrri.Variables {
		stepRes := false
		walkVarName = v.name
		actual, _ := variable.GetVariableValue(ctx, v.name)
		if v.value != nil {
			stepRes = *v.value == actual
		}

		if v.regexPattern != nil {
			stepRes = v.regexPattern.MatchString(actual)
		}

		if stepRes {
			if v.model == OR {
				// fast abort when match the current variable
				result = true
				break
			}
		} else {
			if v.model == AND {
				// fast abort when not match current variable
				result = false
				break
			}
		}

	}

	if result {
		log.DefaultLogger.Infof(RouterLogFormat, "variable rotue rule", "match success", walkVarName)
		return vrri
	}

	log.DefaultLogger.Errorf(RouterLogFormat, "variable rotue rule", "failed match", vrri.Variables, walkVarName)
	return nil
}

type VariableMatchItem struct {
	name         string
	value        *string
	regexPattern *regexp.Regexp
	model        Model
}

func ParseToVariableMatchItem(matcher v2.VariableMatcher) *VariableMatchItem {
	vmi := &VariableMatchItem{
		name:  matcher.Name,
		model: AND,
	}
	vmi.name = matcher.Name
	if matcher.Value != "" {
		vmi.value = &matcher.Value
	}
	if matcher.Regex != "" {
		regPattern, err := regexp.Compile(matcher.Regex)
		if err != nil {
			log.DefaultLogger.Errorf(RouterLogFormat, "variable router rule", "ParseToVariableMatchItem", err)
			return nil
		}
		vmi.regexPattern = regPattern
	}
	if matcher.Model != "" {
		vmi.model = Model(strings.ToLower(matcher.Model))
		if vmi.model != AND && vmi.model != OR {
			log.DefaultLogger.Errorf(RouterLogFormat, "variable router rule", "ParseToVariableMatchItem", "Model is not support", matcher.Model)
			return nil
		}
	}
	return vmi
}
