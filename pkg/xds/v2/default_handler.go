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
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/xds/conv"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// default type url mosn will handle
const (
	EnvoyListener              = "type.googleapis.com/envoy.api.v2.Listener"
	EnvoyCluster               = "type.googleapis.com/envoy.api.v2.Cluster"
	EnvoyClusterLoadAssignment = "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"
	EnvoyRouteConfiguration    = "type.googleapis.com/envoy.api.v2.RouteConfiguration"
)

func init() {
	RegisterTypeURLHandleFunc(EnvoyListener, HandleEnvoyListener)
	RegisterTypeURLHandleFunc(EnvoyCluster, HandleEnvoyCluster)
	RegisterTypeURLHandleFunc(EnvoyClusterLoadAssignment, HandleEnvoyClusterLoadAssignment)
	RegisterTypeURLHandleFunc(EnvoyRouteConfiguration, HandleEnvoyRouteConfiguration)
}

// HandleEnvoyListener parse envoy data to mosn listener config
func HandleEnvoyListener(client *ADSClient, resp *envoy_api_v2.DiscoveryResponse) {
	log.DefaultLogger.Tracef("get lds resp,handle it")
	listeners := client.V2Client.handleListenersResp(resp)
	log.DefaultLogger.Infof("get %d listeners from LDS", len(listeners))
	conv.ConvertAddOrUpdateListeners(listeners)
	if err := client.V2Client.reqRoutes(client.StreamClient); err != nil {
		log.DefaultLogger.Warnf("send thread request rds fail!auto retry next period")
	}
}

// HandleEnvoyCluster parse envoy data to mosn cluster config
func HandleEnvoyCluster(client *ADSClient, resp *envoy_api_v2.DiscoveryResponse) {
	log.DefaultLogger.Tracef("get cds resp,handle it")
	clusters := client.V2Client.handleClustersResp(resp)
	log.DefaultLogger.Infof("get %d clusters from CDS", len(clusters))
	conv.ConvertUpdateClusters(clusters)
	clusterNames := make([]string, 0)

	for _, cluster := range clusters {
		if cluster.Type == envoy_api_v2.Cluster_EDS {
			clusterNames = append(clusterNames, cluster.Name)
		}
	}

	if len(clusterNames) != 0 {
		if err := client.V2Client.reqEndpoints(client.StreamClient, clusterNames); err != nil {
			log.DefaultLogger.Warnf("send thread request eds fail!auto retry next period")
		}
	} else {
		if err := client.V2Client.reqListeners(client.StreamClient); err != nil {
			log.DefaultLogger.Warnf("send thread request lds fail!auto retry next period")
		}
	}
}

// HandleEnvoyClusterLoadAssignment parse envoy data to mosn endpoint config
func HandleEnvoyClusterLoadAssignment(client *ADSClient, resp *envoy_api_v2.DiscoveryResponse) {
	log.DefaultLogger.Tracef("get eds resp,handle it ")
	endpoints := client.V2Client.handleEndpointsResp(resp)
	log.DefaultLogger.Infof("get %d endpoints from EDS", len(endpoints))
	conv.ConvertUpdateEndpoints(endpoints)

	if err := client.V2Client.reqListeners(client.StreamClient); err != nil {
		log.DefaultLogger.Warnf("send thread request lds fail!auto retry next period")
	}
}

// HandleEnvoyRouteConfiguration parse envoy data to mosn route config
func HandleEnvoyRouteConfiguration(client *ADSClient, resp *envoy_api_v2.DiscoveryResponse) {
	log.DefaultLogger.Tracef("get rds resp,handle it")
	routes := client.V2Client.handleRoutesResp(resp)
	log.DefaultLogger.Infof("get %d routes from RDS", len(routes))
	conv.ConvertAddOrUpdateRouters(routes)
}
