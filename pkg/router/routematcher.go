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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	RegisteRouterConfigFactory(protocol.SofaRPC, NewRouteMatcher)
	RegisteRouterConfigFactory(protocol.HTTP2, NewRouteMatcher)
	RegisteRouterConfigFactory(protocol.HTTP1, NewRouteMatcher)
	RegisteRouterConfigFactory(protocol.Xprotocol, NewRouteMatcher)
}

func NewRouteMatcher(config interface{}) (types.Routers, error) {
	routerMatcher := &RouteMatcher{
		virtualHosts:                make(map[string]types.VirtualHost),
		wildcardVirtualHostSuffixes: make(map[int]map[string]types.VirtualHost),
	}

	if config, ok := config.(*v2.Proxy); ok {

		for _, virtualHost := range config.VirtualHosts {

			//todo 补充virtual host 其他成员
			vh := NewVirtualHostImpl(virtualHost, config.ValidateClusters)

			for _, domain := range virtualHost.Domains {

				// Note: we use domain in lowercase
				domain = strings.ToLower(domain)

				if domain == "*" {
					if routerMatcher.defaultVirtualHost != nil {
						log.StartLogger.Fatal("Only a single wildcard domain permitted")
					}
					log.StartLogger.Tracef("route matcher default virtual host")
					routerMatcher.defaultVirtualHost = vh

				} else if len(domain) > 1 && "*" == domain[:1] {
					domainMap := map[string]types.VirtualHost{domain[1:]: vh}
					routerMatcher.wildcardVirtualHostSuffixes[len(domain)-1] = domainMap

				} else if _, ok := routerMatcher.virtualHosts[domain]; ok {
					log.StartLogger.Fatal("Only unique values for domains are permitted, get duplicate domain = %s", domain)
				} else {
					routerMatcher.virtualHosts[domain] = vh
				}
			}
		}
	}

	return routerMatcher, nil
}

// A router wrapper used to matches an incoming request headers to a backend cluster
type RouteMatcher struct {
	virtualHosts                map[string]types.VirtualHost // key: host
	defaultVirtualHost          types.VirtualHost
	wildcardVirtualHostSuffixes map[int]map[string]types.VirtualHost
}

// Routing with Virtual Host
func (rm *RouteMatcher) Route(headers map[string]string, randomValue uint64) types.Route {
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

func (rm *RouteMatcher) findVirtualHost(headers map[string]string) types.VirtualHost {
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
func (rm *RouteMatcher) findWildcardVirtualHost(host string) types.VirtualHost {

	// e.g. foo-bar.baz.com will match *-bar.baz.com
	for wildcardLen, wildcardMap := range rm.wildcardVirtualHostSuffixes {
		if wildcardLen >= len(host) {
			continue
		} else {
			for domainKey, virtualHost := range wildcardMap {
				if domainKey == host[len(host)-wildcardLen:] {
					return virtualHost
				}
			}
		}
	}

	return nil
}

func (rm *RouteMatcher) AddRouter(routerName string) {}

func (rm *RouteMatcher) DelRouter(routerName string) {}
