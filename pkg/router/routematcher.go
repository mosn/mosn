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
	"fmt"
	"sort"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	RegisterRouterConfigFactory(protocol.SofaRPC, NewRouteMatcher)
	RegisterRouterConfigFactory(protocol.HTTP2, NewRouteMatcher)
	RegisterRouterConfigFactory(protocol.HTTP1, NewRouteMatcher)
	RegisterRouterConfigFactory(protocol.Xprotocol, NewRouteMatcher)
}

// NewRouteMatcher
// New 'routeMatcher' according to config
func NewRouteMatcher(config interface{}) (types.Routers, error) {
	routerMatcher := &routeMatcher{
		virtualHosts:                             make(map[string]types.VirtualHost),
		wildcardVirtualHostSuffixes:              make(map[int]map[string]types.VirtualHost),
		greaterSortedWildcardVirtualHostSuffixes: []int{},
	}

	if config, ok := config.(*v2.Proxy); ok {
		for _, virtualHost := range config.VirtualHosts {
			// if virtualHost is nil, it is a invalid config, panic in NewVirtualHostImpl
			//if nil == virtualHost {
			//	continue
			//}

			vh, err := NewVirtualHostImpl(virtualHost, config.ValidateClusters)
			if err != nil {
				return nil, err
			}
			for _, domain := range virtualHost.Domains {
				// Note: we use domain in lowercase
				domain = strings.ToLower(domain)

				if domain == "*" {
					if routerMatcher.defaultVirtualHost != nil {
						return nil, fmt.Errorf("Only a single wildcard domain permitted")
					}
					log.StartLogger.Tracef("add route matcher default virtual host")
					routerMatcher.defaultVirtualHost = vh

				} else if len(domain) > 1 && "*" == domain[:1] {
					// first key: wildcard's len
					m, ok := routerMatcher.wildcardVirtualHostSuffixes[len(domain)-1]
					if !ok {
						m = map[string]types.VirtualHost{}
						routerMatcher.wildcardVirtualHostSuffixes[len(domain)-1] = m
					}
					// add check, different from envoy
					// exactly same wildcard domain is unique
					wildcard := domain[1:]
					if _, ok := m[wildcard]; ok {
						return nil, fmt.Errorf("Only unique values for domains are permitted, get duplicate domain = %s", domain)
					}
					m[wildcard] = vh

				} else {
					if _, ok := routerMatcher.virtualHosts[domain]; ok {
						return nil, fmt.Errorf("Only unique values for domains are permitted, get duplicate domain = %s", domain)
					}
					routerMatcher.virtualHosts[domain] = vh
				}
			}
		}
	} else {
		return nil, fmt.Errorf("NewRouteMatcher failure: config is not in type of *v2.Proxy")
	}

	for key := range routerMatcher.wildcardVirtualHostSuffixes {
		routerMatcher.greaterSortedWildcardVirtualHostSuffixes = append(routerMatcher.greaterSortedWildcardVirtualHostSuffixes, key)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(routerMatcher.greaterSortedWildcardVirtualHostSuffixes)))

	return routerMatcher, nil
}

// A router wrapper used to matches an incoming request headers to a backend cluster
type routeMatcher struct {
	virtualHosts                map[string]types.VirtualHost // key: host
	defaultVirtualHost          types.VirtualHost
	wildcardVirtualHostSuffixes map[int]map[string]types.VirtualHost
	// array member is the lens of the wildcard in descending order
	// used for longest match
	greaterSortedWildcardVirtualHostSuffixes []int
}

// Routing with Virtual Host
func (rm *routeMatcher) Route(headers map[string]string, randomValue uint64) types.Route {
	// First Step: Select VirtualHost with "host" in Headers form VirtualHost Array
	log.StartLogger.Tracef("routing header = %v,randomValue=%v", headers, randomValue)
	virtualHost := rm.findVirtualHost(headers)

	if virtualHost == nil {
		log.DefaultLogger.Errorf("No VirtualHost Found when Routing, Request Headers = %+v", headers)
		return nil
	}

	// Second Step: Match Route from Routes in a Virtual Host
	routerInstance := virtualHost.GetRouteFromEntries(headers, randomValue)

	if routerInstance == nil {
		log.DefaultLogger.Errorf("No Router Instance Found when Routing, Request Headers = %+v", headers)
	}

	return routerInstance
}

func (rm *routeMatcher) findVirtualHost(headers map[string]string) types.VirtualHost {
	if len(rm.virtualHosts) == 0 && rm.defaultVirtualHost != nil {
		log.StartLogger.Tracef("route matcher find virtual host return default virtual host")
		return rm.defaultVirtualHost
	}

	host := strings.ToLower(headers[strings.ToLower(protocol.MosnHeaderHostKey)])

	// for service, header["host"] == header["service"] == servicename
	// or use only a unique key for sofa's virtual host
	if virtualHost, ok := rm.virtualHosts[host]; ok {
		return virtualHost
	}

	if len(rm.wildcardVirtualHostSuffixes) > 0 {

		if vhost := rm.findWildcardVirtualHost(host); vhost != nil {
			return vhost
		}
	}

	return rm.defaultVirtualHost
}

// Rule: longest wildcard suffix match against the host
func (rm *routeMatcher) findWildcardVirtualHost(host string) types.VirtualHost {
	// e.g. foo-bar.baz.com will match *-bar.baz.com
	// foo-bar.baz.com should match *-bar.baz.com before matching *.baz.com
	for _, wildcardLen := range rm.greaterSortedWildcardVirtualHostSuffixes {
		if wildcardLen >= len(host) {
			continue
		} else {
			wildcardMap := rm.wildcardVirtualHostSuffixes[wildcardLen]
			for domainKey, virtualHost := range wildcardMap {
				if domainKey == host[len(host)-wildcardLen:] {
					return virtualHost
				}
			}
		}
	}

	return nil
}

func (rm *routeMatcher) AddRouter(routerName string) {}

func (rm *routeMatcher) DelRouter(routerName string) {}
