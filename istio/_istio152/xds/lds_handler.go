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

func (ads *AdsStreamClient) handleLds(resp *envoy_api_v2.DiscoveryResponse) error {
	listeners := HandleListenerResponse(resp)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("get %d listeners from LDS", len(listeners))
	}
	ads.config.converter.ConvertAddOrUpdateListeners(listeners)

	ads.AckResponse(resp)

	req := CreateRdsRequest(ads.config)
	return ads.streamClient.Send(req)
}

func CreateLdsRequest(config *AdsConfig) *envoy_api_v2.DiscoveryRequest {
	return &envoy_api_v2.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: []string{},
		TypeUrl:       EnvoyListener,
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node:          config.Node(),
	}
}

func HandleListenerResponse(resp *envoy_api_v2.DiscoveryResponse) []*envoy_api_v2.Listener {
	listeners := make([]*envoy_api_v2.Listener, 0, len(resp.Resources))
	for _, res := range resp.Resources {
		listener := &envoy_api_v2.Listener{}
		if err := ptypes.UnmarshalAny(res, listener); err != nil {
			log.DefaultLogger.Errorf("ADSClient unmarshal listener fail: %v", err)
		}
		listeners = append(listeners, listener)
	}
	return listeners

}
