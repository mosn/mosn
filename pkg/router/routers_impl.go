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
	"sort"
	"strings"

	"mosn.io/mosn/pkg/variable"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
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

	virtualHostWithPort VirtualHostWithPort

	// stored all vritual host, same as the config order
	virtualHosts []types.VirtualHost
}

type VirtualHostWithPort struct {
	//store host not contain * config
	//just like: key =>www.test.com  value => {"*":index},{"8080",index}
	virtualHostPortsMap map[string]map[string]int

	//key =>8080  value => [{9,".test.com",index},{4,".com",index}]
	//key =>*     value => [{9,".test.com",index},{4,".com",index}]
	portWildcardVirtualHost map[string][]WildcardVirtualHostWithPort
	// *:* as default
	defaultVirtualHostWithPortIndex int
}

type WildcardVirtualHostWithPort struct {
	hostLen int
	host    string
	index   int
}

type WildcardVirtualHostWithPortSlice []WildcardVirtualHostWithPort

func (a WildcardVirtualHostWithPortSlice) Len() int {
	return len(a)
}
func (a WildcardVirtualHostWithPortSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a WildcardVirtualHostWithPortSlice) Less(i, j int) bool {
	return a[j].hostLen < a[i].hostLen
}

func (ri *routersImpl) MatchRoute(ctx context.Context, headers api.HeaderMap) api.Route {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRoute", headers)
	}
	virtualHost := ri.findVirtualHost(ctx)
	if virtualHost == nil {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRoute", "no virtual host found")
		}
		return nil
	}
	router := virtualHost.GetRouteFromEntries(ctx, headers)
	if router == nil {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRoute", "no route found")
		}
	}
	return router
}

func (ri *routersImpl) MatchAllRoutes(ctx context.Context, headers api.HeaderMap) []api.Route {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchAllRoutes", headers)
	}
	virtualHost := ri.findVirtualHost(ctx)
	if virtualHost == nil {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchAllRoutes", "no virtual host found")
		}
		return nil
	}
	routers := virtualHost.GetAllRoutesFromEntries(ctx, headers)
	if len(routers) == 0 {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchAllRoutes", "no route found")
		}
	}
	return routers
}

func (ri *routersImpl) MatchRouteFromHeaderKV(ctx context.Context, headers api.HeaderMap, key string, value string) api.Route {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRouteFromHeaderKV", headers)
	}
	virtualHost := ri.findVirtualHost(ctx)
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
	rb, err := NewRouteBase(vh, route)
	if err != nil {
		msg := fmt.Sprintf("new RouteBase failed, domain: %s, error: %v", domain, err)
		log.DefaultLogger.Errorf(RouterLogFormat, "routers", "AddRoute", msg)
		return -1
	}
	if err := vh.AddRoute(rb); err != nil {
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
	// if host contain port  ,use  findVirtualHostWithPortIndex
	if strings.Contains(host, ":") {
		return ri.findVirtualHostWithPortIndex(host)
	} else {
		if len(ri.virtualHostsIndex) > 0 {
			if index, ok := ri.virtualHostsIndex[host]; ok {
				return index
			}
		}
		if len(ri.wildcardVirtualHostSuffixesIndex) > 0 {
			if index := ri.findWildcardVirtualHost(host); index != -1 {
				return index
			}
		}
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "findVirtualHost", "found default virtual host only")
		}
		return ri.defaultVirtualHostIndex
	}
}

func (ri *routersImpl) findVirtualHostWithPortIndex(host string) int {
	//charIndex always >= 0
	charIndex := strings.Index(host, ":")
	pureHost := host[:charIndex]
	purePort := host[charIndex+1:]

	//match host and port
	// www.test.com:8080  is before www.test.com:*   return
	// www.test.com:*     is before *.test.com:8080  return
	// *.test.com:8080    is before *.com:8080       return
	// *.com:8080         is before *.com:*          return
	// *.com:*            is before *:*              return
	// *:*                is before *                return
	if len(ri.virtualHostWithPort.virtualHostPortsMap) > 0 {
		portMap, ok := ri.virtualHostWithPort.virtualHostPortsMap[pureHost]
		if ok {
			defaultIndex := -1
			for port, index := range portMap {
				if purePort == port {
					return index
				}
				if port == "*" {
					defaultIndex = index
				}
			}
			if defaultIndex != -1 {
				return defaultIndex
			}
		}
	}

	if len(ri.virtualHostWithPort.portWildcardVirtualHost) > 0 {
		//try to match  *.test.com:8080
		wildcardVirtualHostPortArray, ok := ri.virtualHostWithPort.portWildcardVirtualHost[purePort]
		if ok {
			for _, wildcardVirtualHostPort := range wildcardVirtualHostPortArray {
				if wildcardVirtualHostPort.hostLen >= len(pureHost) {
					continue
				}
				hostIndex := len(pureHost) - wildcardVirtualHostPort.hostLen
				if wildcardVirtualHostPort.host == pureHost[hostIndex:] {
					return wildcardVirtualHostPort.index
				}
			}
		}

		//try to match  *.com:*
		wildcardVirtualHostPortArray, ok = ri.virtualHostWithPort.portWildcardVirtualHost["*"]
		if ok {
			for _, wildcardVirtualHostPort := range wildcardVirtualHostPortArray {
				if wildcardVirtualHostPort.hostLen >= len(pureHost) {
					continue
				}
				//找到* 的位置，然后做截取 判断 是否相等
				hostIndex := len(pureHost) - wildcardVirtualHostPort.hostLen
				if wildcardVirtualHostPort.host == pureHost[hostIndex:] {
					return wildcardVirtualHostPort.index
				}
			}
		}
	}

	// *:* return before *
	if ri.virtualHostWithPort.defaultVirtualHostWithPortIndex >= 0 {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "findVirtualHostWithPortIndex", "found default virtual host port only")
		}
		return ri.virtualHostWithPort.defaultVirtualHostWithPortIndex
	}

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "findVirtualHostWithPortIndex", "found default virtual host only")
	}
	//return default *
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

func (ri *routersImpl) findVirtualHost(ctx context.Context) types.VirtualHost {
	hostHeader, err := variable.GetVariableValue(ctx, types.VarHost)
	index := -1
	if err == nil && hostHeader != "" {
		//we use domain in lowercase
		host := strings.ToLower(hostHeader)
		index = ri.findVirtualHostIndex(host)
	}
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
		virtualHostWithPort: VirtualHostWithPort{
			virtualHostPortsMap:     make(map[string]map[string]int),
			portWildcardVirtualHost: make(map[string][]WildcardVirtualHostWithPort),
		},
	}
	configImpl := NewConfigImpl(routerConfig)
	for index, vhConfig := range routerConfig.VirtualHosts {
		vh, err := NewVirtualHostImpl(&vhConfig)
		if err != nil {
			return nil, err
		}

		routers.virtualHosts = append(routers.virtualHosts, vh)
		vh.globalRouteConfig = configImpl
		for _, domain := range vhConfig.Domains {
			domain = strings.ToLower(domain) // we use domain in lowercase
			//split contain port's config and not contain port‘s config
			if strings.Index(domain, ":") < 0 {
				err = generateHostConfig(domain, index, routers)
				if err != nil {
					return nil, err
				}
			} else {
				err = generateHostWithPortConfig(domain, index, routers)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	for key := range routers.wildcardVirtualHostSuffixesIndex {
		routers.greaterSortedWildcardVirtualHostSuffixes = append(routers.greaterSortedWildcardVirtualHostSuffixes, key)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(routers.greaterSortedWildcardVirtualHostSuffixes)))

	// port's config sort by host len
	if len(routers.virtualHostWithPort.portWildcardVirtualHost) > 0 {
		for _, value := range routers.virtualHostWithPort.portWildcardVirtualHost {
			sort.Sort(WildcardVirtualHostWithPortSlice(value))
		}
	}
	return routers, nil
}

func generateHostConfig(domain string, index int, routers *routersImpl) error {
	if domain == "*" {
		if routers.defaultVirtualHostIndex != -1 {
			log.DefaultLogger.Errorf(RouterLogFormat, "routers", "NewRouters", "duplicate default virtualhost")
			return ErrDuplicateVirtualHost
		}
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof(RouterLogFormat, "routers", "NewRouters", "add route matcher default virtual host")
		}
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
			return ErrDuplicateVirtualHost
		}
		m[wildcard] = index
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof(RouterLogFormat, "routers", "NewRouters", "add router domain: "+domain)
		}
	} else {
		if _, ok := routers.virtualHostsIndex[domain]; ok {
			log.DefaultLogger.Errorf(RouterLogFormat, "routers", "NewRouters", "only unique values for domains are permitted, domain:"+domain)
			return ErrDuplicateVirtualHost
		}
		routers.virtualHostsIndex[domain] = index
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof(RouterLogFormat, "routers", "NewRouters", "add router domain: "+domain)
		}
	}
	return nil
}

func generateHostWithPortConfig(domain string, index int, routers *routersImpl) error {

	charIndex := strings.Index(domain, ":")
	if charIndex < 0 {
		return nil
	}

	host := domain[:charIndex]
	port := domain[charIndex+1:]

	if host == "" || strings.Contains(host, ":") {
		log.DefaultLogger.Errorf(RouterLogFormat, "routers", "generateHostWithPortConfig", "virtual host is invalid, domain:"+domain)
		return ErrNoVirtualHost
	}

	if port == "" || strings.Contains(port, ":") {
		log.DefaultLogger.Errorf(RouterLogFormat, "routers", "generateHostWithPortConfig", "virtual host port is invalid, domain:"+domain)
		return ErrNoVirtualHostPort
	}

	if domain == "*:*" {
		routers.virtualHostWithPort.defaultVirtualHostWithPortIndex = index
	} else if !strings.Contains(host, "*") {
		m, ok := routers.virtualHostWithPort.virtualHostPortsMap[host]
		if !ok {
			m = map[string]int{}
			m[port] = index
			routers.virtualHostWithPort.virtualHostPortsMap[host] = m
		} else {
			_, ok := m[port]
			if ok {
				log.DefaultLogger.Errorf(RouterLogFormat, "routers", "generateHostWithPortConfig", "only unique values for host:port are permitted, domain:"+domain)
				return ErrDuplicateHostPort
			}
			m[port] = index
		}
	} else if len(host) > 0 && "*" == host[:1] {

		if strings.Contains(host[1:], "*") {
			log.DefaultLogger.Errorf(RouterLogFormat, "routers", "generateHostWithPortConfig", "only unique wildcard domain permitted, domain:"+domain)
			return ErrDuplicateVirtualHost
		}

		newWildcardVirtualHostWithPort := WildcardVirtualHostWithPort{
			hostLen: len(host) - 1,
			host:    host[1:],
			index:   index,
		}
		m, ok := routers.virtualHostWithPort.portWildcardVirtualHost[port]
		if !ok {
			tempArray := []WildcardVirtualHostWithPort{newWildcardVirtualHostWithPort}
			routers.virtualHostWithPort.portWildcardVirtualHost[port] = tempArray
		} else {
			m = append(m, newWildcardVirtualHostWithPort)
			routers.virtualHostWithPort.portWildcardVirtualHost[port] = m
		}
	} else {
		log.DefaultLogger.Errorf(RouterLogFormat, "routers", "generateHostWithPortConfig", "duplicate virtual host port, domain:"+domain)
		return ErrDuplicateHostPort
	}

	return nil
}
