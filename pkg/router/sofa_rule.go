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
	"sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/common/log"
	"sofastack.io/sofa-mosn/pkg/types"
)

func SofaRouterFactory(headers []v2.HeaderMatcher) RouteBase {
	for _, header := range headers {
		if header.Name == types.SofaRouteMatchKey {
			return &SofaRouteRuleImpl{
				matchName:  header.Name,
				matchValue: header.Value,
			}
		}
	}
	log.DefaultLogger.Errorf(RouterLogFormat, "sofa router factory", "create failed", headers)
	return nil
}

type SofaRouteRuleImpl struct {
	*RouteRuleImplBase
	matchName  string
	matchValue string
}

func (srri *SofaRouteRuleImpl) PathMatchCriterion() types.PathMatchCriterion {
	return srri
}

func (srri *SofaRouteRuleImpl) RouteRule() types.RouteRule {
	return srri
}

func (srri *SofaRouteRuleImpl) Matcher() string {
	return srri.matchValue
}

func (srri *SofaRouteRuleImpl) MatchType() types.PathMatchType {
	return types.SofaHeader
}

func (srri *SofaRouteRuleImpl) FinalizeRequestHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {
}

func (srri *SofaRouteRuleImpl) Match(headers types.HeaderMap, randomValue uint64) types.Route {
	if value, ok := headers.Get(types.SofaRouteMatchKey); ok {
		if value == srri.matchValue || srri.matchValue == ".*" {
			return srri
		}
	}
	log.DefaultLogger.Errorf(RouterLogFormat, "sofa rotue rule", "failed match", headers)
	return nil
}
