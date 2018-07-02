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
	"time"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type V2Client struct {
	ServiceCluster string
	ServiceNode    string
	Config         *XDSConfig
}

type XDSConfig struct {
	ADSConfig *ADSConfig
	Clusters  map[string]*ClusterConfig
}

type ClusterConfig struct {
	LbPolicy       xdsapi.Cluster_LbPolicy
	Address        []string
	ConnectTimeout *time.Duration
}

type ADSConfig struct {
	ApiType      core.ApiConfigSource_ApiType
	RefreshDelay *time.Duration
	Services     []*ServiceConfig
	StreamClient *StreamClient
}

type ADSClient struct {
	AdsConfig       *ADSConfig
	StreamClient    ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	V2Client        *V2Client
	MosnConfig      *config.MOSNConfig
	SendControlChan chan int
	RecvControlChan chan int
	StopChan        chan int
}

type ServiceConfig struct {
	Timeout       *time.Duration
	ClusterConfig *ClusterConfig
}

type StreamClient struct {
	Client ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	Conn   *grpc.ClientConn
	Cancel context.CancelFunc
}
