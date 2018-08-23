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
	"errors"
	"regexp"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/markphelps/optional"
)

func NewVirtualHostImpl(virtualHost *v2.VirtualHost, validateClusters bool) (*VirtualHostImpl, error) {
	var virtualHostImpl = &VirtualHostImpl{virtualHostName: virtualHost.Name}

	switch virtualHost.RequireTLS {
	case "EXTERNALONLY":
		virtualHostImpl.sslRequirements = types.EXTERNALONLY
	case "ALL":
		virtualHostImpl.sslRequirements = types.ALL
	default:
		virtualHostImpl.sslRequirements = types.NONE
	}

	for _, route := range virtualHost.Routers {
		routeRuleImplBase, err := NewRouteRuleImplBase(virtualHostImpl, &route)
		if err != nil {
			return nil, err
		}

		if route.Match.Prefix != "" {
			virtualHostImpl.routes = append(virtualHostImpl.routes, &PrefixRouteRuleImpl{
				routeRuleImplBase,
				route.Match.Prefix,
			})

		} else if route.Match.Path != "" {
			virtualHostImpl.routes = append(virtualHostImpl.routes, &PathRouteRuleImpl{
				routeRuleImplBase,
				route.Match.Path,
			})

		} else if route.Match.Regex != "" {
			if regPattern, err := regexp.Compile(route.Match.Regex); err == nil {
				virtualHostImpl.routes = append(virtualHostImpl.routes, &RegexRouteRuleImpl{
					routeRuleImplBase,
					route.Match.Regex,
					*regPattern,
				})
			} else {
				log.DefaultLogger.Errorf("Compile Regex Error")
			}
		} else {
			// todo delete hack
			// hack here to do sofa's routing policy
			for _, header := range route.Match.Headers {
				if header.Name == types.SofaRouteMatchKey {
					virtualHostImpl.routes = append(virtualHostImpl.routes, &SofaRouteRuleImpl{
						RouteRuleImplBase: routeRuleImplBase,
						matchValue:        header.Value,
					})
				}
			}
		}
	}
	if len(virtualHostImpl.routes) == 0 {
		return nil, errors.New("routes must specify one of prefix/path/regex/header")
	}

	// todo check cluster's validity
	if validateClusters {
	}

	// Add Virtual Cluster
	for _, vc := range virtualHost.VirtualClusters {

		if regxPattern, err := regexp.Compile(vc.Pattern); err == nil {
			virtualHostImpl.virtualClusters = append(virtualHostImpl.virtualClusters,
				VirtualClusterEntry{
					name:    vc.Name,
					method:  optional.NewString(vc.Method),
					pattern: *regxPattern,
				})
		} else {
			log.DefaultLogger.Errorf("Compile Error")
		}
	}

	return virtualHostImpl, nil
}

type VirtualHostImpl struct {
	virtualHostName       string
	routes                []RouteBase //route impl
	virtualClusters       []VirtualClusterEntry
	sslRequirements       types.SslRequirements
	corsPolicy            types.CorsPolicy
	globalRouteConfig     *configImpl
	requestHeadersParser  *headerParser
	responseHeadersParser *headerParser
}

func (vh *VirtualHostImpl) Name() string {

	return vh.virtualHostName
}

func (vh *VirtualHostImpl) CorsPolicy() types.CorsPolicy {

	return nil
}

func (vh *VirtualHostImpl) RateLimitPolicy() types.RateLimitPolicy {

	return nil
}

func (vh *VirtualHostImpl) GetRouteFromEntries(headers map[string]string, randomValue uint64) types.Route {
	// todo check tls
	for _, route := range vh.routes {
		if routeEntry := route.Match(headers, randomValue); routeEntry != nil {
			return routeEntry
		}
	}

	return nil
}

type VirtualClusterEntry struct {
	pattern regexp.Regexp
	method  optional.String
	name    string
}

func (vce *VirtualClusterEntry) VirtualClusterName() string {

	return vce.name
}
