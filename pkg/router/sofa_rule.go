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

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

type SofaRouteRuleImpl struct {
	*RouteRuleImplBase
	fastmatch string
}

func (srri *SofaRouteRuleImpl) PathMatchCriterion() api.PathMatchCriterion {
	return srri
}

func (srri *SofaRouteRuleImpl) RouteRule() api.RouteRule {
	return srri
}

func (srri *SofaRouteRuleImpl) Matcher() string {
	return srri.fastmatch
}

func (srri *SofaRouteRuleImpl) MatchType() api.PathMatchType {
	return api.SofaHeader
}

func (srri *SofaRouteRuleImpl) FinalizeRequestHeaders(ctx context.Context, headers api.HeaderMap, requestInfo api.RequestInfo) {
	srri.RouteRuleImplBase.FinalizeRequestHeaders(ctx, headers, requestInfo)
}

func (srri *SofaRouteRuleImpl) Match(ctx context.Context, headers api.HeaderMap) api.Route {
	if srri.fastmatch == "" {
		if srri.matchRoute(ctx, headers) {
			return srri
		}
	} else {
		value, _ := headers.Get(types.SofaRouteMatchKey)
		if value != "" {
			if value == srri.fastmatch || srri.fastmatch == ".*" {
				return srri
			}
		}
	}
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, RouterLogFormat, "sofa rotue rule", "failed match, macther %s", srri.fastmatch)
	}
	return nil
}
