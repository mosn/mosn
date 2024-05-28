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
	"net"
	"sort"
	"strings"

	"mosn.io/pkg/variable"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// An implementation of types.Routers
type routersImpl struct {
	defaultVirtualHostIndex int
	//store definite host config
	//key =>www.test.com  value => {"*",index},{"8080",index}
	virtualHostPortsMap map[string]map[string]int

	//key =>8080  value => [{9,".test.com",index},{4,".com",index}]
	//key =>*     value => [{9,".test.com",index},{4,".com",index}]
	portWildcardVirtualHost map[string][]WildcardVirtualHostWithPort

	// stored all virtual host, same as the config order
	virtualHosts []types.VirtualHost
}

type WildcardVirtualHostWithPort struct {
	hostLen int
	host    string
	index   int
}

// for easy to sort
type WildcardVirtualHostWithPortSlice []WildcardVirtualHostWithPort

func (a WildcardVirtualHostWithPortSlice) Len() int {
	return len(a)
}
func (a WildcardVirtualHostWithPortSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// realization is not the same as normal less, simplify the reverse op
func (a WildcardVirtualHostWithPortSlice) Less(i, j int) bool {
	return a[j].hostLen < a[i].hostLen
}

func (ri *routersImpl) MatchRoute(ctx context.Context, headers api.HeaderMap) api.Route {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRoute", "")
		log.DefaultLogger.Tracef(RouterLogFormat, "routers", "MatchRoute", headers)
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
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchAllRoutes", "")
		log.DefaultLogger.Tracef(RouterLogFormat, "routers", "MatchAllRoutes", headers)
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
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "MatchRouteFromHeaderKV", "")
		log.DefaultLogger.Tracef(RouterLogFormat, "routers", "MatchRouteFromHeaderKV", headers)
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

func (ri *routersImpl) findVirtualHost(ctx context.Context) types.VirtualHost {
	// optimize, if there is only a default, use it
	if len(ri.virtualHostPortsMap) == 0 && len(ri.portWildcardVirtualHost) == 0 &&
		ri.defaultVirtualHostIndex != -1 {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routers", "findVirtualHost", "found default virtual host only")
		}
		return ri.virtualHosts[ri.defaultVirtualHostIndex]
	}
	hostHeader, err := variable.GetString(ctx, types.VarHost)
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

func (ri *routersImpl) findVirtualHostIndex(hostPort string) int {
	host, port, err := splitHostPortGraceful(hostPort)
	if err != nil {
		return -1
	}
	return ri.findHighestPriorityIndex(host, port)
}

// match priority rules:
// priority 1: definite host and definite port (No port belong to this )
// priority 2: definite host and wildcard port
// priority 3: wildcard host and definite port
// priority 4: wildcard host and wildcard port
// priority 5: default
func (ri *routersImpl) findHighestPriorityIndex(host, port string) int {

	if len(ri.virtualHostPortsMap) > 0 {
		portMap, ok := ri.virtualHostPortsMap[host]
		if ok {
			index, ok := portMap[port]
			//priority 1: definite host and definite port (No port belong to this )
			if ok {
				return index
			}

			//priority 2: definite host and wildcard port
			index, ok = portMap["*"]
			if ok {
				return index
			}
		}
	}

	if len(ri.portWildcardVirtualHost) > 0 {
		//priority 3: wildcard host and definite port
		wildcardVirtualHostPortArray, ok := ri.portWildcardVirtualHost[port]
		if ok {
			for _, wildcardVirtualHostPort := range wildcardVirtualHostPortArray {
				if wildcardVirtualHostPort.hostLen >= len(host) {
					continue
				}
				hostIndex := len(host) - wildcardVirtualHostPort.hostLen
				if wildcardVirtualHostPort.host == host[hostIndex:] {
					return wildcardVirtualHostPort.index
				}
			}
		}

		// priority 4: wildcard host and wildcard port
		wildcardVirtualHostPortArray, ok = ri.portWildcardVirtualHost["*"]
		if ok {
			for _, wildcardVirtualHostPort := range wildcardVirtualHostPortArray {
				if wildcardVirtualHostPort.hostLen >= len(host) {
					continue
				}
				hostIndex := len(host) - wildcardVirtualHostPort.hostLen
				if wildcardVirtualHostPort.host == host[hostIndex:] {
					return wildcardVirtualHostPort.index
				}
			}
		}
	}

	//priority 5: default
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "routers", "findVirtualHostWithPortIndex", "found default virtual host only")
	}
	return ri.defaultVirtualHostIndex
}

// NewRouters creates a types.Routers by according to config
func NewRouters(routerConfig *v2.RouterConfiguration) (types.Routers, error) {
	if routerConfig == nil || len(routerConfig.VirtualHosts) == 0 {
		log.DefaultLogger.Infof(RouterLogFormat, "routers", "NewRouters", fmt.Sprintf("router config is %v", routerConfig))
		return nil, ErrNilRouterConfig
	}
	routers := &routersImpl{
		defaultVirtualHostIndex: -1, // not exists
		virtualHosts:            []types.VirtualHost{},
		virtualHostPortsMap:     make(map[string]map[string]int),
		portWildcardVirtualHost: make(map[string][]WildcardVirtualHostWithPort),
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

			host, port, err := splitHostPortGraceful(domain)
			if err != nil {
				return nil, ErrNoVirtualHostPort
			}

			err = routers.generateHostWithPortConfig(host, port, index, routers)
			if err != nil {
				return nil, err
			}
		}
	}

	// port's config sort by host len
	if len(routers.portWildcardVirtualHost) > 0 {
		for _, value := range routers.portWildcardVirtualHost {
			sort.Sort(WildcardVirtualHostWithPortSlice(value))
		}
	}
	return routers, nil
}
func (ri *routersImpl) generateHostWithPortConfig(host, port string, index int, routers *routersImpl) error {
	if host == "" && port == "" {
		log.DefaultLogger.Errorf(RouterLogFormat, "routers", "generateHostWithPortConfig", "virtual host is invalid, host and port is null ")
		return ErrNoVirtualHost
	}
	//set default
	if host == "*" && (port == "*" || port == "") {
		if routers.defaultVirtualHostIndex != -1 {
			log.DefaultLogger.Errorf(RouterLogFormat, "routers", "NewRouters", "duplicate default virtualhost")
			return ErrDuplicateVirtualHost
		}
		routers.defaultVirtualHostIndex = index
		return nil
	}

	//for host definite match
	if !strings.Contains(host, "*") {
		m, ok := routers.virtualHostPortsMap[host]
		if !ok {
			m = map[string]int{}
			m[port] = index
			routers.virtualHostPortsMap[host] = m
		} else {
			_, ok := m[port]
			if ok {
				log.DefaultLogger.Errorf(RouterLogFormat, "routers", "generateHostWithPortConfig", "only unique values for host:port are permitted, host:port "+host+":"+port)
				return ErrDuplicateHostPort
			}
			m[port] = index
		}
		return nil
	}

	//for wildcard host match
	if len(host) > 0 && "*" == host[:1] {
		newWildcardVirtualHostWithPort := WildcardVirtualHostWithPort{
			hostLen: len(host) - 1,
			host:    host[1:],
			index:   index,
		}
		m, ok := routers.portWildcardVirtualHost[port]
		if !ok {
			tempArray := []WildcardVirtualHostWithPort{newWildcardVirtualHostWithPort}
			routers.portWildcardVirtualHost[port] = tempArray
		} else {
			for _, tempM := range m {
				if tempM.host == newWildcardVirtualHostWithPort.host {
					log.DefaultLogger.Errorf(RouterLogFormat, "routers", "generateHostWithPortConfig", "only unique wildcard domain permitted, host:port "+host+":"+port)
					return ErrDuplicateVirtualHost
				}
			}
			m = append(m, newWildcardVirtualHostWithPort)
			routers.portWildcardVirtualHost[port] = m
		}
		return nil
	}

	log.DefaultLogger.Errorf(RouterLogFormat, "routers", "generateHostWithPortConfig", "virtual host port is invalid, host:port "+host+":"+port)
	return ErrNoVirtualHostPort
}

// "missing port in address" is not error
func splitHostPortGraceful(hostPort string) (host, port string, err error) {
	host, port, err = net.SplitHostPort(hostPort)
	if err != nil {
		if addrErr, ok := err.(*net.AddrError); ok && addrErr.Err == "missing port in address" {
			return hostPort, port, nil
		} else {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				msg := fmt.Sprintf("host invalid : %s, error: %v", hostPort, err)
				log.DefaultLogger.Debugf(RouterLogFormat, "routers", "SplitHostPortGraceful", msg)
			}
			return "", "", err
		}
	}
	return host, port, nil
}
