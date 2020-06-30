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
	"fmt"
	"sort"
	"strings"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

// An implementation of types.Routers
type routersImpl struct {
	// virtual host index
	virtualHostsIndex                map[string]int
	defaultVirtualHostIndex          int
	wildcardVirtualHostSuffixesIndex map[int]map[string]int
	//  array member is the lens of the wildcard in descending order
	//  used for longest match
	greaterSortedWildcardVirtualHostSuffixes []int
	// stored all vritual host, same as the config order
	virtualHosts []types.VirtualHost
}

func (ri *routersImpl) MatchRoute(headers api.HeaderMap, randomValue uint64) api.Route {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRoute", headers)
	}
	virtualHost := ri.findVirtualHost(headers)
	if virtualHost == nil {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRoute", "no virtual host found")
		}
		return nil
	}
	router := virtualHost.GetRouteFromEntries(headers, randomValue)
	if router == nil {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRoute", "no route found")
		}
	}
	return router
}

func (ri *routersImpl) MatchAllRoutes(headers api.HeaderMap, randomValue uint64) []api.Route {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchAllRoutes", headers)
	}
	virtualHost := ri.findVirtualHost(headers)
	if virtualHost == nil {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchAllRoutes", "no virtual host found")
		}
		return nil
	}
	routers := virtualHost.GetAllRoutesFromEntries(headers, randomValue)
	if len(routers) == 0 {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchAllRoutes", "no route found")
		}
	}
	return routers
}

func (ri *routersImpl) MatchRouteFromHeaderKV(headers api.HeaderMap, key string, value string) api.Route {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRouteFromHeaderKV", headers)
	}
	virtualHost := ri.findVirtualHost(headers)
	if virtualHost == nil {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRouteFromHeaderKV", "no virtual host found")
		}
		return nil
	}
	router := virtualHost.GetRouteFromHeaderKV(key, value)
	if router == nil {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRouteFromHeaderKV", "no route found")
		}
	}
	return router
}

// AddRoute adds a route into virtual host
// find virtual host by domain
// returns the virtualhost index, -1 means no virtual host found
func (ri *routersImpl) AddRoute(domain string, route *v2.Router) int {
	index := ri.findVirtualHostIndex(domain)
	if index == -1 {
		log.DefaultLogger.Errorf(RouterLogFormat, "routers", "AddRoute", "no virtual host found from domain: "+domain)
		return index
	}
	vh := ri.virtualHosts[index]
	if err := vh.AddRoute(route); err != nil {
		msg := fmt.Sprintf("add route into virtualhost failed, domain: %s, error: %v", domain, err)
		log.DefaultLogger.Errorf(RouterLogFormat, "routers", "AddRoute", msg)
		return -1
	}
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof(RouterLogFormat, "routers", "AddRoute", "adds a new route into domain: "+domain)
	}
	return index
}

func (ri *routersImpl) RemoveAllRoutes(domain string) int {
	index := ri.findVirtualHostIndex(domain)
	if index == -1 {
		log.DefaultLogger.Errorf(RouterLogFormat, "routers", "RemoveAllRoutes", "no virtual host found from domain: "+domain)
		return index
	}
	vh := ri.virtualHosts[index]
	vh.RemoveAllRoutes()
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof(RouterLogFormat, "routers", "RemoveAllRoutes", "clear all routes in domain: "+domain)
	}
	return index
}

func (ri *routersImpl) findVirtualHostIndex(host string) int {
	if index, ok := ri.virtualHostsIndex[host]; ok {
		return index
	}
	if len(ri.wildcardVirtualHostSuffixesIndex) > 0 {
		if index := ri.findWildcardVirtualHost(host); index != -1 {
			return index
		}
	}
	return ri.defaultVirtualHostIndex
}

func (ri *routersImpl) findWildcardVirtualHost(host string) int {
	// longest wildcard suffix match against the host
	// e.g. foo-bar.baz.com will match *-bar.baz.com
	// foo-bar.baz.com should match *-bar.baz.com before matching *.baz.com
	for _, wildcardLen := range ri.greaterSortedWildcardVirtualHostSuffixes {
		if wildcardLen >= len(host) {
			continue
		}
		wildcardMap := ri.wildcardVirtualHostSuffixesIndex[wildcardLen]
		hostIndex := len(host) - wildcardLen
		for domain, index := range wildcardMap {
			if domain == host[hostIndex:] {
				return index
			}
		}
	}
	return -1
}

func (ri *routersImpl) findVirtualHost(headers api.HeaderMap) types.VirtualHost {
	// optimize, if there is only a default, use it
	if len(ri.virtualHostsIndex) == 0 &&
		len(ri.wildcardVirtualHostSuffixesIndex) == 0 &&
		ri.defaultVirtualHostIndex != -1 {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "findVirtualHost", "found default virtual host only")
		}
		return ri.virtualHosts[ri.defaultVirtualHostIndex]
	}
	//we use domain in lowercase
	key := strings.ToLower(protocol.MosnHeaderHostKey)
	hostHeader, _ := headers.Get(key)
	host := strings.ToLower(hostHeader)
	index := ri.findVirtualHostIndex(host)
	if index == -1 {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "findVirtualHost", "no virtual host found")
		}
		return nil
	}
	return ri.virtualHosts[index]
}

// NewRouters creates a types.Routers by according to config
func NewRouters(routerConfig *v2.RouterConfiguration) (types.Routers, error) {
	if routerConfig == nil || len(routerConfig.VirtualHosts) == 0 {
		log.DefaultLogger.Infof(RouterLogFormat, "routers", "NewRouters", fmt.Sprintf("router config is %v", routerConfig))
		return nil, ErrNilRouterConfig
	}
	routers := &routersImpl{
		virtualHostsIndex:                        make(map[string]int),
		defaultVirtualHostIndex:                  -1, // not exists
		wildcardVirtualHostSuffixesIndex:         make(map[int]map[string]int),
		greaterSortedWildcardVirtualHostSuffixes: []int{},
		virtualHosts:                             []types.VirtualHost{},
	}
	configImpl := NewConfigImpl(routerConfig)
	for index, vhConfig := range routerConfig.VirtualHosts {
		vh, err := NewVirtualHostImpl(vhConfig)
		if err != nil {
			return nil, err
		}

		routers.virtualHosts = append(routers.virtualHosts, vh)
		vh.globalRouteConfig = configImpl
		for _, domain := range vhConfig.Domains {
			domain = strings.ToLower(domain) // we use domain in lowercase
			if domain == "*" {
				if routers.defaultVirtualHostIndex != -1 {
					log.DefaultLogger.Errorf(RouterLogFormat, "routers", "NewRouters", "duplicate default virtualhost")
					return nil, ErrDuplicateVirtualHost
				}
				log.DefaultLogger.Infof(RouterLogFormat, "routers", "NewRouters", "add route matcher default virtual host")
				routers.defaultVirtualHostIndex = index
			} else if len(domain) > 1 && "*" == domain[:1] {
				m, ok := routers.wildcardVirtualHostSuffixesIndex[len(domain)-1]
				if !ok {
					m = map[string]int{}
					routers.wildcardVirtualHostSuffixesIndex[len(domain)-1] = m
				}
				// add check, different from envoy
				// exactly same wildcard domain is unique
				wildcard := domain[1:]
				if _, ok := m[wildcard]; ok {
					log.DefaultLogger.Errorf(RouterLogFormat, "routers", "NewRouters", "only unique wildcard domain permitted, domain:"+domain)
					return nil, ErrDuplicateVirtualHost
				}
				m[wildcard] = index
				log.DefaultLogger.Infof(RouterLogFormat, "routers", "NewRouters", "add router domain: "+domain)
			} else {
				if _, ok := routers.virtualHostsIndex[domain]; ok {
					log.DefaultLogger.Errorf(RouterLogFormat, "routers", "NewRouters", "only unique values for domains are permitted, domain:"+domain)
					return nil, ErrDuplicateVirtualHost
				}
				routers.virtualHostsIndex[domain] = index
				log.DefaultLogger.Infof(RouterLogFormat, "routers", "NewRouters", "add router domain: "+domain)
			}
		}
	}
	for key := range routers.wildcardVirtualHostSuffixesIndex {
		routers.greaterSortedWildcardVirtualHostSuffixes = append(routers.greaterSortedWildcardVirtualHostSuffixes, key)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(routers.greaterSortedWildcardVirtualHostSuffixes)))
	return routers, nil
}
