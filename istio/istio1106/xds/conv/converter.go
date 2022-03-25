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

package conv

import (
	"sync"

	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

// Converter as an interface for mock test
type Converter interface {
	Stats() *XdsStats
	AppendRouterName(name string)
	GetRouterNames() []string
	ConvertAddOrUpdateRouters(routers []*envoy_config_route_v3.RouteConfiguration)
	ConvertAddOrUpdateListeners(listeners []*envoy_config_listener_v3.Listener)
	ConvertDeleteListeners(listeners []*envoy_config_listener_v3.Listener)
	ConvertUpdateClusters(clusters []*envoy_config_cluster_v3.Cluster)
	ConvertDeleteClusters(clusters []*envoy_config_cluster_v3.Cluster)
	ConvertUpdateEndpoints(loadAssignments []*envoy_config_endpoint_v3.ClusterLoadAssignment) error
}

type xdsConverter struct {
	stats XdsStats
	// rdsrecords stores the router config from router discovery
	rdsrecords map[string]struct{}
	mu         sync.Mutex
}

func NewConverter() *xdsConverter {
	return &xdsConverter{
		stats:      NewXdsStats(),
		rdsrecords: map[string]struct{}{},
	}
}

func (cvt *xdsConverter) Stats() *XdsStats {
	return &cvt.stats
}

func (cvt *xdsConverter) AppendRouterName(name string) {
	cvt.mu.Lock()
	defer cvt.mu.Unlock()
	cvt.rdsrecords[name] = struct{}{}
}

func (cvt *xdsConverter) GetRouterNames() []string {
	cvt.mu.Lock()
	defer cvt.mu.Unlock()
	names := make([]string, 0, len(cvt.rdsrecords))
	for name := range cvt.rdsrecords {
		names = append(names, name)
	}
	return names
}
