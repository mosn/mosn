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

func (ads *AdsStreamClient) handleEds(resp *envoy_api_v2.DiscoveryResponse) error {
	endpoints := HandleEndpointResponse(resp)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("get %d endpoints from EDS", len(endpoints))
	}
	ads.config.converter.ConvertUpdateEndpoints(endpoints)

	ads.AckResponse(resp)

	// lds
	req := CreateLdsRequest(ads.config)
	return ads.streamClient.Send(req)
}

func CreateEdsRequest(config *AdsConfig, clusterNames []string) *envoy_api_v2.DiscoveryRequest {
	return &envoy_api_v2.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: clusterNames,
		TypeUrl:       EnvoyClusterLoadAssignment,
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node:          config.Node(),
	}
}

func HandleEndpointResponse(resp *envoy_api_v2.DiscoveryResponse) []*envoy_api_v2.ClusterLoadAssignment {
	lbAssignments := make([]*envoy_api_v2.ClusterLoadAssignment, 0, len(resp.Resources))
	for _, res := range resp.Resources {
		lbAssignment := &envoy_api_v2.ClusterLoadAssignment{}
		if err := ptypes.UnmarshalAny(res, lbAssignment); err != nil {
			log.DefaultLogger.Errorf("ADSClient unmarshal lbAssignment fail: %v", err)
		}
		lbAssignments = append(lbAssignments, lbAssignment)
	}
	return lbAssignments
}
