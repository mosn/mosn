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
	"sync"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

type VirtualHostImpl struct {
	virtualHostName       string
	mutex                 sync.RWMutex
	routes                []RouteBase
	fastIndex             map[string]map[string]api.Route
	globalRouteConfig     *configImpl
	requestHeadersParser  *headerParser
	responseHeadersParser *headerParser
	perFilterConfig       map[string]interface{}
}

func (vh *VirtualHostImpl) Name() string {
	return vh.virtualHostName
}

func (vh *VirtualHostImpl) addRouteBase(route api.RouteBase) {
	if route == nil {
		return
	}
	vh.mutex.Lock()
	defer vh.mutex.Unlock()
	vh.routes = append(vh.routes, route)
	// make fast index, used in certain scenarios
	// TODO: rule can be extended
	hmc := route.RouteRule().HeaderMatchCriteria()
	if hmc != nil && hmc.Len() == 1 && hmc.Get(0).MatchType() == api.ValueExact {
		key := hmc.Get(0).Key()
		value := hmc.Get(0).Matcher()
		valueMap, ok := vh.fastIndex[key]
		if !ok {
			valueMap = make(map[string]api.Route)
			vh.fastIndex[key] = valueMap
		}
		valueMap[value] = route
	}
	return

}

func (vh *VirtualHostImpl) GetRouteFromEntries(ctx context.Context, headers api.HeaderMap) api.Route {
	vh.mutex.RLock()
	defer vh.mutex.RUnlock()
	for _, route := range vh.routes {
		if routeEntry := route.Match(ctx, headers); routeEntry != nil {
			return routeEntry
		}
	}
	return nil
}

func (vh *VirtualHostImpl) GetAllRoutesFromEntries(ctx context.Context, headers api.HeaderMap) []api.Route {
	vh.mutex.RLock()
	defer vh.mutex.RUnlock()
	var routes []api.Route
	for _, route := range vh.routes {
		if r := route.Match(ctx, headers); r != nil {
			routes = append(routes, r)
		}
	}
	return routes
}

func (vh *VirtualHostImpl) GetRouteFromHeaderKV(key, value string) api.Route {
	vh.mutex.RLock()
	defer vh.mutex.RUnlock()
	if m, ok := vh.fastIndex[key]; ok {
		if r, ok := m[value]; ok {
			return r
		}
	}
	return nil
}

// AddRoute always returns nil, keep api.VirtualHost interface compatible
func (vh *VirtualHostImpl) AddRoute(route api.RouteBase) error {
	vh.addRouteBase(route)
	return nil
}

func (vh *VirtualHostImpl) RemoveAllRoutes() {
	vh.mutex.Lock()
	defer vh.mutex.Unlock()
	// clear the value map
	vh.fastIndex = make(map[string]map[string]api.Route)
	// clear the routes
	vh.routes = vh.routes[:0]
	return
}

func (vh *VirtualHostImpl) PerFilterConfig() map[string]interface{} {
	return vh.perFilterConfig
}

func (vh *VirtualHostImpl) FinalizeRequestHeaders(ctx context.Context, headers api.HeaderMap, requestInfo api.RequestInfo) {
	vh.requestHeadersParser.evaluateHeaders(ctx, headers)
	vh.globalRouteConfig.requestHeadersParser.evaluateHeaders(ctx, headers)
}

func (vh *VirtualHostImpl) FinalizeResponseHeaders(ctx context.Context, headers api.HeaderMap, requestInfo api.RequestInfo) {
	vh.responseHeadersParser.evaluateHeaders(ctx, headers)
	vh.globalRouteConfig.responseHeadersParser.evaluateHeaders(ctx, headers)
}

// NewVirtualHostImpl convert mosn VirtualHost config to actual virtual host object
func NewVirtualHostImpl(virtualHost *v2.VirtualHost) (*VirtualHostImpl, error) {
	vhImpl := &VirtualHostImpl{
		virtualHostName:       virtualHost.Name,
		fastIndex:             make(map[string]map[string]api.Route),
		requestHeadersParser:  getHeaderParser(virtualHost.RequestHeadersToAdd, virtualHost.RequestHeadersToRemove),
		responseHeadersParser: getHeaderParser(virtualHost.ResponseHeadersToAdd, virtualHost.ResponseHeadersToRemove),
		perFilterConfig:       virtualHost.PerFilterConfig,
	}
	for _, route := range virtualHost.Routers {
		rb, err := NewRouteBase(vhImpl, &route)
		if err != nil {
			return nil, err
		}
		vhImpl.addRouteBase(rb)
	}
	return vhImpl, nil
}

func NewRouteBase(vh api.VirtualHost, route *v2.Router) (api.RouteBase, error) {
	base, err := NewRouteRuleImplBase(vh, route)
	if err != nil {
		log.DefaultLogger.Errorf(RouterLogFormat, "virtualhost", "NewRouteBase", err)
		return nil, err
	}
	var router RouteBase
	if route.Match.Prefix != "" {
		router = &PrefixRouteRuleImpl{
			BaseHTTPRouteRule: NewBaseHTTPRouteRule(base, route.Match.Headers),
			prefix:            route.Match.Prefix,
		}
	} else if route.Match.Path != "" {
		router = &PathRouteRuleImpl{
			BaseHTTPRouteRule: NewBaseHTTPRouteRule(base, route.Match.Headers),
			path:              route.Match.Path,
		}
	} else if route.Match.Regex != "" {
		regPattern, err := regexp.Compile(route.Match.Regex)
		if err != nil {
			log.DefaultLogger.Errorf(RouterLogFormat, "virtualhost", "NewRouteBase", err)
			return nil, err
		}
		router = &RegexRouteRuleImpl{
			BaseHTTPRouteRule: NewBaseHTTPRouteRule(base, route.Match.Headers),
			regexStr:          route.Match.Regex,
			regexPattern:      regPattern,
		}
	} else if len(route.Match.Variables) > 0 {
		variableRouter := &VariableRouteRuleImpl{
			RouteRuleImplBase: base,
			Variables:         make([]*VariableMatchItem, len(route.Match.Variables)),
		}
		for i := range route.Match.Variables {
			variableRouter.Variables[i] = ParseToVariableMatchItem(route.Match.Variables[i])
		}
		router = variableRouter
	} else if len(route.Match.DslExpressions) > 0 {
		dslRouter := &DslExpressionRouteRuleImpl{
			RouteRuleImplBase:  base,
			originalExpression: route.Match.DslExpressions,
			DslExpressions:     parseConfigToDslExpression(route.Match.DslExpressions),
		}
		router = dslRouter
	} else {
		router = CreateRPCRule(base, route.Match.Headers)
	}
	return router, nil
}
