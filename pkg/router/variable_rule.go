/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package router

import (
	"context"
	"regexp"
	"strings"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/variable"
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

func (vrri *VariableRouteRuleImpl) HeaderMatchCriteria() api.KeyValueMatchCriteria {
	return nil
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

func (vrri *VariableRouteRuleImpl) FinalizeRequestHeaders(ctx context.Context, headers api.HeaderMap, requestInfo api.RequestInfo) {
}

func (vrri *VariableRouteRuleImpl) Match(ctx context.Context, headers api.HeaderMap) api.Route {
	result := true
	walkVarName := ""
	lastMode := AND
	for _, v := range vrri.Variables {
		walkVarName = v.name
		curStepRes := false
		actual, _ := variable.GetString(ctx, v.name)
		if v.value != nil {
			curStepRes = *v.value == actual
		}

		if v.regexPattern != nil {
			curStepRes = v.regexPattern.MatchString(actual)
		}
		// AND 则需要与上之前的结果
		if lastMode == AND {
			result = result && curStepRes
		} else {
			// 上一步mode是or,则重新计算结果
			result = curStepRes
		}
		if result {
			if v.model == OR {
				// fast abort when match the current variable
				break
			}
		}
		lastMode = v.model
	}

	if result {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "variable route rule", "match success", walkVarName)
		}
		return vrri
	}

	log.DefaultLogger.Errorf(RouterLogFormat, "variable route rule", "failed match", vrri.Variables, walkVarName)
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
