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

package v2

import (
	"errors"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/xds/v2/rds"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core1 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
)

func (c *ClientV2) reqRoutes(streamClient ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient) error {
	if streamClient == nil {
		return errors.New("stream client is nil")
	}
	routerNames := rds.GetRouterNames()
	if len(routerNames) < 1 {
		log.DefaultLogger.Tracef("0 routers, skip rds request")
		return nil
	}
	log.DefaultLogger.Tracef("routers to subcriber: %+v", routerNames)
	err := streamClient.Send(&envoy_api_v2.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: routerNames,
		TypeUrl:       "type.googleapis.com/envoy.api.v2.RouteConfiguration",
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node: &envoy_api_v2_core1.Node{
			Id:      c.ServiceNode,
			Cluster: c.ServiceCluster,
		},
	})
	if err != nil {
		log.DefaultLogger.Fatalf("get routers fail: %v", err)
		return err
	}
	return nil
}

func (c *ClientV2) handleRoutesResp(resp *envoy_api_v2.DiscoveryResponse) []*envoy_api_v2.RouteConfiguration {
	routes := make([]*envoy_api_v2.RouteConfiguration, 0)
	for _, res := range resp.Resources {
		route := envoy_api_v2.RouteConfiguration{}
		route.Unmarshal(res.GetValue())
		routes = append(routes, &route)
	}
	return routes
}
