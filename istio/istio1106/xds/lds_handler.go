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
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"mosn.io/mosn/pkg/log"
)

func (ads *AdsStreamClient) handleLds(resp *envoy_service_discovery_v3.DiscoveryResponse) error {
	listeners := HandleListenerResponse(resp)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("get %d listeners from LDS", len(listeners))
	}
	ads.config.converter.ConvertAddOrUpdateListeners(listeners)
	info := &responseInfo{
		ResponseNonce: resp.Nonce,
		VersionInfo:   resp.VersionInfo,
		ResourceNames: []string{},
	}
	ads.config.previousInfo.Store(EnvoyListener, info)

	ads.AckResponse(resp)

	req := CreateRdsRequest(ads.config)
	return ads.streamClient.Send(req)
}

func CreateLdsRequest(config *AdsConfig) *envoy_service_discovery_v3.DiscoveryRequest {
	info := config.previousInfo.Find(EnvoyListener)
	return &envoy_service_discovery_v3.DiscoveryRequest{
		VersionInfo:   info.VersionInfo,
		ResourceNames: info.ResourceNames,
		TypeUrl:       EnvoyListener,
		ResponseNonce: info.ResponseNonce,
		ErrorDetail:   nil,
		Node:          config.Node(),
	}
}

func HandleListenerResponse(resp *envoy_service_discovery_v3.DiscoveryResponse) []*envoy_config_listener_v3.Listener {
	listeners := make([]*envoy_config_listener_v3.Listener, 0, len(resp.Resources))
	for _, res := range resp.Resources {
		listener := &envoy_config_listener_v3.Listener{}
		if err := ptypes.UnmarshalAny(res, listener); err != nil {
			log.DefaultLogger.Errorf("ADSClient unmarshal listener fail: %v", err)
			continue
		}
		listeners = append(listeners, listener)
	}
	return listeners
}
