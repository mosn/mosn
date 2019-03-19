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

// Package router, VirtualHost Config rules:
// 1. A VirtualHost should have one or more Domains, or it will be ignore
// 2. A VirtualHost has a group of Router (Routers), the first successful matched is used. Notice that the order of router
// 3. priority: domain > '*' > wildcard-domain

// From https://www.envoyproxy.io/docs/envoy/latest/api-v1/route_config/vhost

// A list of domains (host/authority header) that will be matched to this virtual host.
// Wildcard hosts are supported in the form of “*.foo.com” or “*-bar.foo.com”.
// Note that the wildcard will not match the empty string. e.g. “*-bar.foo.com” will match “baz-bar.foo.com” but not “-bar.foo.com”.
// Additionally, a special entry “*” is allowed which will match any host/authority header.
// Only a single virtual host in the entire route configuration can match on “*”.
// A domain must be unique across all virtual hosts or the config will fail to load.
// We do a longest wildcard suffix match against the host that's passed in.
// (e.g. foo-bar.baz.com should match *-bar.baz.com before matching *.baz.com)

package router

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// A router wrapper used to matches an incoming request headers to a backend cluster
type routeMatcher struct {
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

// NewRouteMatcher creates a types.Routers by according to config
func NewRouteMatcher(routerConfig *v2.RouterConfiguration) (types.Routers, error) {
	if routerConfig == nil || len(routerConfig.VirtualHosts) == 0 {
		return nil, fmt.Errorf("create router matcher error, empty config: %v", routerConfig)
	}
	routerMatcher := &routeMatcher{
		virtualHostsIndex:                        make(map[string]int),
		defaultVirtualHostIndex:                  -1, // not exists
		wildcardVirtualHostSuffixesIndex:         make(map[int]map[string]int),
		greaterSortedWildcardVirtualHostSuffixes: []int{},
		virtualHosts:                             []types.VirtualHost{},
	}
	configImpl := NewConfigImpl(routerConfig)

	for index, vhConfig := range routerConfig.VirtualHosts {
		vh, err := NewVirtualHostImpl(vhConfig, false)
		if err != nil {
			return nil, err
		}
		// store the virtual hosts
		routerMatcher.virtualHosts = append(routerMatcher.virtualHosts, vh)
		vh.globalRouteConfig = configImpl
		for _, domain := range vhConfig.Domains {
			domain = strings.ToLower(domain) // we use domain in lowercase
			if domain == "*" {
				log.DefaultLogger.Tracef("add route matcher default virtual host")
				if routerMatcher.defaultVirtualHostIndex != -1 {
					return nil, errors.New("create router matcher error,  only a single wildcard domain permitted")
				}
				routerMatcher.defaultVirtualHostIndex = index
			} else if len(domain) > 1 && "*" == domain[:1] {
				log.DefaultLogger.Tracef("add route matcher wildcard domain: %s", domain)
				// first key: wildcard's len
				m, ok := routerMatcher.wildcardVirtualHostSuffixesIndex[len(domain)-1]
				if !ok {
					m = map[string]int{}
					routerMatcher.wildcardVirtualHostSuffixesIndex[len(domain)-1] = m
				}
				// add check, different from envoy
				// exactly same wildcard domain is unique
				wildcard := domain[1:]
				if _, ok := m[wildcard]; ok {
					return nil, errors.New("create router matcher error,  only unique wildcard domain permitted")
				}
				m[wildcard] = index
			} else {
				log.DefaultLogger.Tracef("add route matcher domain: %s", domain)
				if _, ok := routerMatcher.virtualHostsIndex[domain]; ok {
					return nil, errors.New("create router matcher error, only unique values for domains are permitted")
				}
				routerMatcher.virtualHostsIndex[domain] = index
			}
		}
	}
	for key := range routerMatcher.wildcardVirtualHostSuffixesIndex {
		routerMatcher.greaterSortedWildcardVirtualHostSuffixes = append(routerMatcher.greaterSortedWildcardVirtualHostSuffixes, key)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(routerMatcher.greaterSortedWildcardVirtualHostSuffixes)))
	return routerMatcher, nil
}

// MatchRoute returns the first route that matched
func (rm *routeMatcher) MatchRoute(headers types.HeaderMap, randomValue uint64) types.Route {
	log.DefaultLogger.Tracef("routing header = %v", headers)
	virtualHost := rm.findVirtualHost(headers)
	if virtualHost == nil {
		log.DefaultLogger.Tracef("no virtual host found")
		return nil
	}
	router := virtualHost.GetRouteFromEntries(headers, randomValue)
	if router == nil {
		log.DefaultLogger.Tracef("no router found")
	}
	return router
}

// MatchAllRoutes returns all route that matched
func (rm *routeMatcher) MatchAllRoutes(headers types.HeaderMap, randomValue uint64) []types.Route {
	log.DefaultLogger.Tracef("routing header = %v", headers)
	virtualHost := rm.findVirtualHost(headers)
	if virtualHost == nil {
		log.DefaultLogger.Tracef("no virtual host found")
		return nil
	}
	routers := virtualHost.GetAllRoutesFromEntries(headers, randomValue)
	if len(routers) == 0 {
		log.DefaultLogger.Tracef("no router found")
	}
	return routers
}

func (rm *routeMatcher) MatchRouteFromHeaderKV(headers types.HeaderMap, key string, value string) types.Route {
	log.DefaultLogger.Tracef("routing header = %v", headers)
	virtualHost := rm.findVirtualHost(headers)
	if virtualHost == nil {
		log.DefaultLogger.Tracef("no virtual host found")
		return nil
	}
	router := virtualHost.GetRouteFromHeaderKV(key, value)
	if router == nil {
		log.DefaultLogger.Tracef("no router found")
	}
	return router
}

// AddRoute adds a route into virtual host
// find virtual host by domain
// returns the virtualhost index, -1 means no virtual host found
func (rm *routeMatcher) AddRoute(domain string, route *v2.Router) int {
	index := rm.findVirtualHostIndex(domain)
	if index == -1 {
		log.DefaultLogger.Tracef("no virtual host found")
		return index
	}
	vh := rm.virtualHosts[index]
	if err := vh.AddRoute(route); err != nil {
		log.DefaultLogger.Errorf("add route into virtual host failed: %v", err)
		return -1
	}
	return index
}

func (rm *routeMatcher) RemoveAllRoutes(domain string) int {
	index := rm.findVirtualHostIndex(domain)
	if index == -1 {
		log.DefaultLogger.Tracef("no virtual host found")
		return index
	}
	vh := rm.virtualHosts[index]
	vh.RemoveAllRoutes()
	return index
}

func (rm *routeMatcher) findVirtualHost(headers types.HeaderMap) types.VirtualHost {
	// optimize, if there is only a default, use it
	if len(rm.virtualHostsIndex) == 0 &&
		len(rm.wildcardVirtualHostSuffixesIndex) == 0 &&
		rm.defaultVirtualHostIndex != -1 {
		log.DefaultLogger.Tracef("found default virtual host only")
		return rm.virtualHosts[rm.defaultVirtualHostIndex]
	}
	//we use domain in lowercase
	key := strings.ToLower(protocol.MosnHeaderHostKey)
	hostHeader, _ := headers.Get(key)
	host := strings.ToLower(hostHeader)
	index := rm.findVirtualHostIndex(host)
	if index == -1 {
		log.DefaultLogger.Tracef("no virtual host found")
		return nil

	}
	return rm.virtualHosts[index]
}

func (rm *routeMatcher) findVirtualHostIndex(host string) int {
	//  for sofa service, header["host"] == header["service"] == servicename
	if index, ok := rm.virtualHostsIndex[host]; ok {
		return index
	}
	if len(rm.wildcardVirtualHostSuffixesIndex) > 0 {
		if index := rm.findWildcardVirtualHost(host); index != -1 {
			return index
		}
	}
	return rm.defaultVirtualHostIndex
}

// Rule: longest wildcard suffix match against the host
func (rm *routeMatcher) findWildcardVirtualHost(host string) int {
	// e.g. foo-bar.baz.com will match *-bar.baz.com
	// foo-bar.baz.com should match *-bar.baz.com before matching *.baz.com
	for _, wildcardLen := range rm.greaterSortedWildcardVirtualHostSuffixes {
		if wildcardLen >= len(host) {
			continue
		}
		wildcardMap := rm.wildcardVirtualHostSuffixesIndex[wildcardLen]
		for domain, index := range wildcardMap {
			if domain == host[len(host)-wildcardLen:] {
				return index
			}
		}
	}
	return -1
}
