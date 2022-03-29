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

package xds

import (
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"mosn.io/mosn/pkg/log"
)

func (ads *AdsStreamClient) handleRds(resp *envoy_service_discovery_v3.DiscoveryResponse) error {
	routes := HandleRouteResponse(resp)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("get %d routes from RDS", len(routes))
	}
	ads.config.converter.ConvertAddOrUpdateRouters(routes)
	resourceNames := make([]string, 0, len(routes))
	for _, rt := range routes {
		resourceNames = append(resourceNames, rt.Name)
	}
	info := &responseInfo{
		ResponseNonce: resp.Nonce,
		VersionInfo:   resp.VersionInfo,
		ResourceNames: resourceNames,
	}
	ads.config.previousInfo.Store(EnvoyRoute, info)
	ads.AckResponse(resp)

	return nil
}

func CreateRdsRequest(config *AdsConfig) *envoy_service_discovery_v3.DiscoveryRequest {
	routerNames := config.converter.GetRouterNames()
	if len(routerNames) < 1 {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("no routers, skip rds request")
		}
		return nil
	}
	info := config.previousInfo.Find(EnvoyRoute)
	return &envoy_service_discovery_v3.DiscoveryRequest{
		VersionInfo:   info.VersionInfo,
		ResourceNames: routerNames,
		TypeUrl:       EnvoyRoute,
		ResponseNonce: info.ResponseNonce,
		ErrorDetail:   nil,
		Node:          config.Node(),
	}
}

func HandleRouteResponse(resp *envoy_service_discovery_v3.DiscoveryResponse) []*envoy_config_route_v3.RouteConfiguration {
	routes := make([]*envoy_config_route_v3.RouteConfiguration, 0, len(resp.Resources))
	for _, res := range resp.Resources {
		route := &envoy_config_route_v3.RouteConfiguration{}
		if err := ptypes.UnmarshalAny(res, route); err != nil {
			log.DefaultLogger.Errorf("ADSClient unmarshal route fail: %v", err)
			continue
		}
		routes = append(routes, route)
	}
	return routes
}
