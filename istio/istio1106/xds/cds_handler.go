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
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"mosn.io/mosn/pkg/log"
)

func (ads *AdsStreamClient) handleCds(resp *envoy_service_discovery_v3.DiscoveryResponse) error {
	clusters := HandleClusterResponse(resp)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("get %d clusters from CDS", len(clusters))
	}
	// TODO: handle error, support error detail for ack
	ads.config.converter.ConvertUpdateClusters(clusters)
	// save response info
	ads.config.previousInfo.Store(resp.TypeUrl, &responseInfo{
		ResponseNonce: resp.Nonce,
		VersionInfo:   resp.VersionInfo,
		ResourceNames: []string{}, // CDS ResourcesNames keeps empty
	})
	ads.AckResponse(resp)
	clusterNames := make([]string, 0, len(clusters))
	for _, cluster := range clusters {
		if cluster.GetType() == envoy_config_cluster_v3.Cluster_EDS {
			clusterNames = append(clusterNames, cluster.Name)
		}
	}
	if len(clusterNames) != 0 { // EDS
		ads.config.previousInfo.SetResourceNames(EnvoyEndpoint, clusterNames)
		req := CreateEdsRequest(ads.config)
		return ads.streamClient.Send(req)
	}
	// LDS
	req := CreateLdsRequest(ads.config)
	return ads.streamClient.Send(req)

}

func CreateCdsRequest(config *AdsConfig) *envoy_service_discovery_v3.DiscoveryRequest {
	info := config.previousInfo.Find(EnvoyCluster)
	return &envoy_service_discovery_v3.DiscoveryRequest{
		VersionInfo:   info.VersionInfo,
		ResourceNames: info.ResourceNames,
		TypeUrl:       EnvoyCluster,
		ResponseNonce: info.ResponseNonce,
		ErrorDetail:   nil,
		Node:          config.Node(),
	}
}

func HandleClusterResponse(resp *envoy_service_discovery_v3.DiscoveryResponse) []*envoy_config_cluster_v3.Cluster {
	clusters := make([]*envoy_config_cluster_v3.Cluster, 0, len(resp.Resources))
	for _, res := range resp.Resources {
		cluster := &envoy_config_cluster_v3.Cluster{}
		if err := ptypes.UnmarshalAny(res, cluster); err != nil {
			log.DefaultLogger.Errorf("ADSClient unmarshal cluster fail: %v", err)
			continue
		}
		clusters = append(clusters, cluster)
	}
	return clusters
}
