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
	"regexp"
	"sync"

	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
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
}

func (vh *VirtualHostImpl) Name() string {
	return vh.virtualHostName
}

func (vh *VirtualHostImpl) addRouteBase(route *v2.Router) error {
	base, err := NewRouteRuleImplBase(vh, route)
	if err != nil {
		log.DefaultLogger.Errorf(RouterLogFormat, "virtualhost", "addRouteBase", err)
		return err
	}
	var router RouteBase
	if route.Match.Prefix != "" {
		router = &PrefixRouteRuleImpl{
			RouteRuleImplBase: base,
			prefix:            route.Match.Prefix,
		}
	} else if route.Match.Path != "" {
		router = &PathRouteRuleImpl{
			RouteRuleImplBase: base,
			path:              route.Match.Path,
		}
	} else if route.Match.Regex != "" {
		regPattern, err := regexp.Compile(route.Match.Regex)
		if err != nil {
			log.DefaultLogger.Errorf(RouterLogFormat, "virtualhost", "addRouteBase", err)
			return err
		}
		router = &RegexRouteRuleImpl{
			RouteRuleImplBase: base,
			regexStr:          route.Match.Regex,
			regexPattern:      regPattern,
		}
	} else {
		if router = defaultRouterRuleFactoryOrder.factory(base, route.Match.Headers); router == nil {
			log.DefaultLogger.Errorf(RouterLogFormat, "virtualhost", "addRouteBase", "create default router failed")
			return ErrRouterFactory
		}
	}
	if router != nil {
		vh.mutex.Lock()
		vh.routes = append(vh.routes, router)
		// make fast index, used in certain scenarios
		// TODO: rule can be extended
		if len(route.Match.Headers) == 1 && !route.Match.Headers[0].Regex {
			key := route.Match.Headers[0].Name
			value := route.Match.Headers[0].Value
			valueMap, ok := vh.fastIndex[key]
			if !ok {
				valueMap = make(map[string]api.Route)
				vh.fastIndex[key] = valueMap
			}
			valueMap[value] = router
		}
		vh.mutex.Unlock()
		log.DefaultLogger.Infof(RouterLogFormat, "virtualhost", "addRouteBase", "add a new route rule")
	} else {
		log.DefaultLogger.Errorf(RouterLogFormat, "virtualhost", "addRouteBase", "add a new route rule failed")
	}
	return nil

}

func (vh *VirtualHostImpl) GetRouteFromEntries(headers api.HeaderMap, randomValue uint64) api.Route {
	vh.mutex.RLock()
	defer vh.mutex.RUnlock()
	for _, route := range vh.routes {
		if routeEntry := route.Match(headers, randomValue); routeEntry != nil {
			return routeEntry
		}
	}
	return nil
}

func (vh *VirtualHostImpl) GetAllRoutesFromEntries(headers api.HeaderMap, randomValue uint64) []api.Route {
	vh.mutex.RLock()
	defer vh.mutex.RUnlock()
	var routes []api.Route
	for _, route := range vh.routes {
		if r := route.Match(headers, randomValue); r != nil {
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

func (vh *VirtualHostImpl) AddRoute(route *v2.Router) error {
	return vh.addRouteBase(route)
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

// NewVirtualHostImpl convert mosn VirtualHost config to actual virtual host object
func NewVirtualHostImpl(virtualHost *v2.VirtualHost) (*VirtualHostImpl, error) {
	vhImpl := &VirtualHostImpl{
		virtualHostName:       virtualHost.Name,
		fastIndex:             make(map[string]map[string]api.Route),
		requestHeadersParser:  getHeaderParser(virtualHost.RequestHeadersToAdd, nil),
		responseHeadersParser: getHeaderParser(virtualHost.ResponseHeadersToAdd, virtualHost.ResponseHeadersToRemove),
	}
	for _, route := range virtualHost.Routers {
		if err := vhImpl.addRouteBase(&route); err != nil {
			return nil, err
		}
	}
	return vhImpl, nil
}
