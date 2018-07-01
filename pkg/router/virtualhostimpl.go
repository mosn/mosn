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

	"github.com/markphelps/optional"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func NewVirtualHostImpl(virtualHost *v2.VirtualHost, validateClusters bool) *VirtualHostImpl {
	var virtualHostImpl = &VirtualHostImpl{virtualHostName: virtualHost.Name}

	switch virtualHost.RequireTls {
	case "EXTERNALONLY":
		virtualHostImpl.sslRequirements = types.EXTERNALONLY
	case "ALL":
		virtualHostImpl.sslRequirements = types.ALL
	default:
		virtualHostImpl.sslRequirements = types.NONE
	}

	for _, route := range virtualHost.Routers {

		if route.Match.Prefix != "" {

			virtualHostImpl.routes = append(virtualHostImpl.routes, &PrefixRouteRuleImpl{
				NewRouteRuleImplBase(virtualHostImpl, &route),
				route.Match.Prefix,
			})

		} else if route.Match.Path != "" {
			virtualHostImpl.routes = append(virtualHostImpl.routes, &PathRouteRuleImpl{
				NewRouteRuleImplBase(virtualHostImpl, &route),
				route.Match.Path,
			})

		} else if route.Match.Regex != "" {

			if regPattern, err := regexp.Compile(route.Match.Prefix); err == nil {
				virtualHostImpl.routes = append(virtualHostImpl.routes, &RegexRouteRuleImpl{
					NewRouteRuleImplBase(virtualHostImpl, &route),
					route.Match.Prefix,
					*regPattern,
				})
			} else {
				log.DefaultLogger.Errorf("Compile Regex Error")
			}
		} else {
			for _, header := range route.Match.Headers {
				if header.Name == types.SofaRouteMatchKey {
					virtualHostImpl.routes = append(virtualHostImpl.routes, &SofaRouteRuleImpl{
						RouteRuleImplBase: NewRouteRuleImplBase(virtualHostImpl, &route),
						matchValue:        header.Value,
					})
				}
			}
		}
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

	return virtualHostImpl
}

type VirtualHostImpl struct {
	virtualHostName       string
	routes                []RouteBase //route impl
	virtualClusters       []VirtualClusterEntry
	sslRequirements       types.SslRequirements
	corsPolicy            types.CorsPolicy
	globalRouteConfig     *ConfigImpl
	requestHeadersParser  *HeaderParser
	responseHeadersParser *HeaderParser
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
