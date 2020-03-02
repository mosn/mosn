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
	"strings"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

// TransparentRouteRuleImpl used to "match path" with "prefix match"
type TransparentRouteRuleImpl struct {
	*RouteRuleImplBase
	Prefix string
}

func NewTransparentRouteImpl() types.Route {
	vHost := &v2.VirtualHost{
		Name: "*",
	}
	virtualHostImpl, err := NewVirtualHostImpl(vHost)
	if err != nil {
		return nil
	}

	virtualHostImpl.globalRouteConfig = &configImpl{}

	prefix := "/"
	route := &v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{Prefix: prefix},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: "transparent_cluster",
				},
			},
		},
	}

	routuRule, _ := NewRouteRuleImplBase(virtualHostImpl, route)
	rr := &TransparentRouteRuleImpl{
		routuRule,
		prefix,
	}

	return rr
}

func (prei *TransparentRouteRuleImpl) PathMatchCriterion() api.PathMatchCriterion {
	return prei
}

func (prei *TransparentRouteRuleImpl) RouteRule() api.RouteRule {
	return prei
}

// types.PathMatchCriterion
func (prei *TransparentRouteRuleImpl) Matcher() string {
	return prei.Prefix
}

func (prei *TransparentRouteRuleImpl) MatchType() api.PathMatchType {
	return api.Prefix
}

// types.RouteRule
// override Base
func (prei *TransparentRouteRuleImpl) FinalizeRequestHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {
	prei.finalizeRequestHeaders(headers, requestInfo)
	prei.finalizePathHeader(headers, prei.Prefix)
}

func (prei *TransparentRouteRuleImpl) Match(headers types.HeaderMap, randomValue uint64) types.Route {
	if prei.matchRoute(headers, randomValue) {
		if headerPathValue, ok := headers.Get(protocol.MosnHeaderPathKey); ok {
			if strings.HasPrefix(headerPathValue, prei.Prefix) {
				return prei
			}
		}
	}
	log.DefaultLogger.Debugf(RouterLogFormat, "prefxi route rule", "failed match", headers)
	return nil
}
