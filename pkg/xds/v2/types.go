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
	"sync"
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	v2 "mosn.io/mosn/pkg/config/v2"
)

// XDSConfig contains ADS config and clusters info
type XDSConfig struct {
	ADSConfig *ADSConfig
	Clusters  map[string]*ClusterConfig
}

// ClusterConfig contains an cluster info from static resources
type ClusterConfig struct {
	LbPolicy       envoy_api_v2.Cluster_LbPolicy
	Address        []string
	ConnectTimeout *time.Duration
	TlsContext     *envoy_api_v2_auth.UpstreamTlsContext
}

// ADSConfig contains ADS config from dynamic resources
type ADSConfig struct {
	APIType      core.ApiConfigSource_ApiType
	RefreshDelay *time.Duration
	Services     []*ServiceConfig
	StreamClient *StreamClient
}

// ADSClient communicated with pilot
type ADSClient struct {
	AdsConfig         *ADSConfig
	StreamClientMutex sync.RWMutex
	StreamClient      ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	MosnConfig        *v2.MOSNConfig
	SendControlChan   chan int
	RecvControlChan   chan int
	StopChan          chan int
}

// ServiceConfig for grpc service
type ServiceConfig struct {
	Timeout       *time.Duration
	ClusterConfig *ClusterConfig
}

// StreamClient is an grpc client
type StreamClient struct {
	Client ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	Conn   *grpc.ClientConn
	Cancel context.CancelFunc
}

// TypeURLHandleFunc is a function that used to parse ads type url data
type TypeURLHandleFunc func(client *ADSClient, resp *envoy_api_v2.DiscoveryResponse)
