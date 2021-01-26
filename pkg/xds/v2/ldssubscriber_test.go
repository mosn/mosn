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
	"runtime/debug"
	"sync"
	"testing"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoy_config_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	xdshttp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

func Test_LdsHandler(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestLdsHandler error: %v \n %s", r, string(debug.Stack()))
		}
	}()

	xdsConfig := XDSConfig{}
	adsClient := &ADSClient{
		AdsConfig:         xdsConfig.ADSConfig,
		StreamClientMutex: sync.RWMutex{},
		StreamClient:      nil,
		SendControlChan:   make(chan int),
		RecvControlChan:   make(chan int),
		StopChan:          make(chan int),
	}

	httpManager := &xdshttp.HttpConnectionManager{
		RouteSpecifier: &envoy_config_v2.HttpConnectionManager_Rds{
			Rds: &envoy_config_v2.Rds{
				RouteConfigName: "test_route",
			},
		},
	}
	httpManagerAny, _ := ptypes.MarshalAny(httpManager)
	filter := listener.Filter{
		Name: "envoy.http_connection_manager",
	}
	filter.ConfigType = &listener.Filter_TypedConfig{TypedConfig: httpManagerAny}
	listener := &envoy_api_v2.Listener{
		Name: "test_listener",
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					&filter,
				},
			},
		},
	}

	listenerRes, _ := ptypes.MarshalAny(listener)
	resp := &envoy_api_v2.DiscoveryResponse{
		TypeUrl:   EnvoyListener,
		Resources: []*any.Any{listenerRes},
	}

	if lds := adsClient.handleListenersResp(resp); lds == nil || len(lds) != 1 {
		t.Error("handleListenersResp failed.")
	}
}
