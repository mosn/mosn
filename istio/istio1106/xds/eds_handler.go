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
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"mosn.io/mosn/pkg/log"
)

func (ads *AdsStreamClient) handleEds(resp *envoy_service_discovery_v3.DiscoveryResponse) error {
	endpoints := HandleEndpointResponse(resp)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("get %d endpoints from EDS", len(endpoints))
	}
	ads.config.converter.ConvertUpdateEndpoints(endpoints)
	info := &responseInfo{
		VersionInfo:   resp.VersionInfo,
		ResponseNonce: resp.Nonce,
		ResourceNames: ads.config.previousInfo.Find(EnvoyEndpoint).ResourceNames,
	}
	ads.config.previousInfo.Store(EnvoyEndpoint, info)

	ads.AckResponse(resp)
	// LDS
	req := CreateLdsRequest(ads.config)
	return ads.streamClient.Send(req)
}

func CreateEdsRequest(config *AdsConfig) *envoy_service_discovery_v3.DiscoveryRequest {
	info := config.previousInfo.Find(EnvoyEndpoint)
	return &envoy_service_discovery_v3.DiscoveryRequest{
		VersionInfo:   info.VersionInfo,
		ResourceNames: info.ResourceNames,
		TypeUrl:       EnvoyEndpoint,
		ResponseNonce: info.ResponseNonce,
		ErrorDetail:   nil,
		Node:          config.Node(),
	}
}

func HandleEndpointResponse(resp *envoy_service_discovery_v3.DiscoveryResponse) []*envoy_config_endpoint_v3.ClusterLoadAssignment {
	lbAssignments := make([]*envoy_config_endpoint_v3.ClusterLoadAssignment, 0, len(resp.Resources))
	for _, res := range resp.Resources {
		lbAssignment := &envoy_config_endpoint_v3.ClusterLoadAssignment{}
		if err := ptypes.UnmarshalAny(res, lbAssignment); err != nil {
			log.DefaultLogger.Errorf("ADSClient unmarshal lbAssignment fail: %v", err)
			continue
		}
		lbAssignments = append(lbAssignments, lbAssignment)
	}
	return lbAssignments
}
