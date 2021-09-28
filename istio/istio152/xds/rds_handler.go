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
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"
	"mosn.io/mosn/pkg/log"
)

func (ads *AdsStreamClient) handleRds(resp *envoy_api_v2.DiscoveryResponse) error {
	routes := HandleRouteResponse(resp)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("get %d routes from RDS", len(routes))
	}
	ads.config.converter.ConvertAddOrUpdateRouters(routes)

	ads.AckResponse(resp)

	return nil
}

func CreateRdsRequest(config *AdsConfig) *envoy_api_v2.DiscoveryRequest {
	routerNames := config.converter.GetRouterNames()
	if len(routerNames) < 1 {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("no routers, skip rds request")
		}
		return nil
	}
	return &envoy_api_v2.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: routerNames,
		TypeUrl:       EnvoyRouteConfiguration,
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node:          config.Node(),
	}
}

func HandleRouteResponse(resp *envoy_api_v2.DiscoveryResponse) []*envoy_api_v2.RouteConfiguration {
	routes := make([]*envoy_api_v2.RouteConfiguration, 0, len(resp.Resources))
	for _, res := range resp.Resources {
		route := &envoy_api_v2.RouteConfiguration{}
		if err := ptypes.UnmarshalAny(res, route); err != nil {
			log.DefaultLogger.Errorf("ADSClient unmarshal route fail: %v", err)
			continue
		}
		routes = append(routes, route)
	}
	return routes
}
