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

func (ads *AdsStreamClient) handleCds(resp *envoy_api_v2.DiscoveryResponse) error {
	clusters := HandleClusterResponse(resp)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("get %d clusters from CDS", len(clusters))
	}
	ads.config.converter.ConvertUpdateClusters(clusters)
	ads.AckResponse(resp)

	clusterNames := make([]string, 0)
	for _, cluster := range clusters {
		if cluster.GetType() == envoy_api_v2.Cluster_EDS {
			clusterNames = append(clusterNames, cluster.Name)
		}
	}

	if len(clusterNames) != 0 {
		// eds
		req := CreateEdsRequest(ads.config, clusterNames)
		return ads.streamClient.Send(req)
	} else {
		// lds
		req := CreateLdsRequest(ads.config)
		return ads.streamClient.Send(req)
	}
}

func CreateCdsRequest(config *AdsConfig) *envoy_api_v2.DiscoveryRequest {
	return &envoy_api_v2.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: []string{},
		TypeUrl:       EnvoyCluster,
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node:          config.Node(),
	}
}

func HandleClusterResponse(resp *envoy_api_v2.DiscoveryResponse) []*envoy_api_v2.Cluster {
	clusters := make([]*envoy_api_v2.Cluster, 0, len(resp.Resources))
	for _, res := range resp.Resources {
		cluster := &envoy_api_v2.Cluster{}
		if err := ptypes.UnmarshalAny(res, cluster); err != nil {
			log.DefaultLogger.Errorf("ADSClient unmarshal cluster fail: %v", err)
			continue
		}
		clusters = append(clusters, cluster)
	}
	return clusters
}
