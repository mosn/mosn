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
	"fmt"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// SofaRule supports only simple headers match. and use fastmatch for compatible old mode
type RPCRouteRuleImpl struct {
	*RouteRuleImplBase
	configHeaders types.HeaderMatcher
	fastmatch     string // compatible field
}

func (srri *RPCRouteRuleImpl) HeaderMatchCriteria() api.KeyValueMatchCriteria {
	if srri.configHeaders != nil {
		return srri.configHeaders.HeaderMatchCriteria()
	}
	return nil
}

func (srri *RPCRouteRuleImpl) PathMatchCriterion() api.PathMatchCriterion {
	return srri
}

func (srri *RPCRouteRuleImpl) RouteRule() api.RouteRule {
	return srri
}

func (srri *RPCRouteRuleImpl) Matcher() string {
	return srri.fastmatch
}

func (srri *RPCRouteRuleImpl) MatchType() api.PathMatchType {
	return api.RPCHeader
}

func (srri *RPCRouteRuleImpl) FinalizeRequestHeaders(ctx context.Context, headers api.HeaderMap, requestInfo api.RequestInfo) {
	srri.RouteRuleImplBase.FinalizeRequestHeaders(ctx, headers, requestInfo)
}

func (srri *RPCRouteRuleImpl) Match(ctx context.Context, headers api.HeaderMap) api.Route {
	if srri.fastmatch == "" {
		if srri.configHeaders.Matches(ctx, headers) {
			return srri
		}
	} else {
		// compatible for old version.
		value, _ := headers.Get(types.RPCRouteMatchKey)
		if value != "" {
			if value == srri.fastmatch || srri.fastmatch == ".*" {
				return srri
			}
		}
	}
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, RouterLogFormat, "Match", "sofa route rule", fmt.Sprintf("failed to match, matcher %s", srri.fastmatch))
	}
	return nil
}

func CreateRPCRule(base *RouteRuleImplBase, headers []v2.HeaderMatcher) RouteBase {
	r := &RPCRouteRuleImpl{
		RouteRuleImplBase: base,
	}
	// compatible for simple sofa rule
	if len(headers) == 1 && headers[0].Name == types.RPCRouteMatchKey {
		r.fastmatch = headers[0].Value
	}
	r.configHeaders = CreateCommonHeaderMatcher(headers)
	return r
}
