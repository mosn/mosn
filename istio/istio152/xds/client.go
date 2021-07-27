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
	"context"
	"errors"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/featuregate"
	"mosn.io/mosn/pkg/istio"
	"mosn.io/mosn/pkg/log"
)

// streamClient is an grpc client
type streamClient struct {
	client ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	conn   *grpc.ClientConn
	cancel context.CancelFunc
}

func (sc *streamClient) Send(req *envoy_api_v2.DiscoveryRequest) error {
	if req == nil {
		return nil
	}
	return sc.client.Send(req)
}

type AdsStreamClient struct {
	streamClient *streamClient
	config       *AdsConfig
}

var _ istio.XdsStreamClient = (*AdsStreamClient)(nil)

func NewAdsStreamClient(c *AdsConfig) (*AdsStreamClient, error) {
	if len(c.Services) == 0 {
		log.DefaultLogger.Errorf("no available ads service")
		return nil, errors.New("no available ads service")
	}
	var endpoint string
	var tlsContext *envoy_api_v2_auth.UpstreamTlsContext
	for _, service := range c.Services {
		if service.ClusterConfig == nil {
			continue
		}
		endpoint, _ = service.ClusterConfig.GetEndpoint()
		if len(endpoint) > 0 {
			tlsContext = service.ClusterConfig.TlsContext
			break
		}
	}
	if len(endpoint) == 0 {
		log.DefaultLogger.Errorf("no available ads endpoint")
		return nil, errors.New("no available ads endpoint")
	}

	sc := &streamClient{}
	if tlsContext == nil || !featuregate.Enabled(featuregate.XdsMtlsEnable) {
		conn, err := grpc.Dial(endpoint, grpc.WithInsecure(), generateDialOption())
		if err != nil {
			log.DefaultLogger.Errorf("grpc dial error: %v", err)
			return nil, err
		}
		log.DefaultLogger.Infof("mosn estab grpc connection to pilot at %v", endpoint)
		sc.conn = conn
	} else {
		creds, err := c.getTLSCreds(tlsContext)
		if err != nil {
			log.DefaultLogger.Errorf("xds-grpc get tls creds fail: err= %v", err)
			return nil, err
		}
		conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(creds), generateDialOption())
		if err != nil {
			log.DefaultLogger.Errorf("grpc dial error: %v", err)
			return nil, err
		}
		log.DefaultLogger.Infof("mosn estab grpc connection to pilot at %v", endpoint)
		sc.conn = conn
	}
	client := ads.NewAggregatedDiscoveryServiceClient(sc.conn)
	ctx, cancel := context.WithCancel(context.Background())
	sc.cancel = cancel
	streamClient, err := client.StreamAggregatedResources(ctx)
	if err != nil {
		log.DefaultLogger.Infof("fail to create stream client: %v", err)
		if sc.conn != nil {
			sc.conn.Close()
		}
		return nil, err
	}
	sc.client = streamClient
	return &AdsStreamClient{
		streamClient: sc,
		config:       c,
	}, nil
}

func (ads *AdsStreamClient) Send(req interface{}) error {
	if ads == nil {
		return errors.New("stream client is nil")
	}
	dr, ok := req.(*envoy_api_v2.DiscoveryRequest)
	if !ok {
		return errors.New("invalid request type")
	}
	return ads.streamClient.client.Send(dr)
}

func (ads *AdsStreamClient) Recv() (interface{}, error) {
	return ads.streamClient.client.Recv()
}

const (
	EnvoyListener              = "type.googleapis.com/envoy.api.v2.Listener"
	EnvoyCluster               = "type.googleapis.com/envoy.api.v2.Cluster"
	EnvoyClusterLoadAssignment = "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"
	EnvoyRouteConfiguration    = "type.googleapis.com/envoy.api.v2.RouteConfiguration"
)

func (ads *AdsStreamClient) HandleResponse(resp interface{}) {
	dresp, ok := resp.(*envoy_api_v2.DiscoveryResponse)
	if !ok {
		log.DefaultLogger.Errorf("invalid response type")
		return
	}
	var err error
	// Ads Handler, overwrite the AdsStreamClient to extends more ads or handle
	switch dresp.TypeUrl {
	case EnvoyCluster:
		err = ads.handleCds(dresp)
	case EnvoyClusterLoadAssignment:
		err = ads.handleEds(dresp)
	case EnvoyListener:
		err = ads.handleLds(dresp)
	case EnvoyRouteConfiguration:
		err = ads.handleRds(dresp)
	default:
		err = errors.New("unsupport type url")
	}
	if err != nil {
		log.DefaultLogger.Errorf("handle typeurl %s failed, error: %v", dresp.TypeUrl, err)
	}
}

func (ads *AdsStreamClient) AckResponse(resp *envoy_api_v2.DiscoveryResponse) {
	ack := &envoy_api_v2.DiscoveryRequest{
		VersionInfo:   resp.VersionInfo,
		ResourceNames: []string{},
		TypeUrl:       resp.TypeUrl,
		ResponseNonce: resp.Nonce,
		ErrorDetail:   nil,
		Node:          ads.config.Node(),
	}
	if err := ads.streamClient.client.Send(ack); err != nil {
		log.DefaultLogger.Errorf("response %s ack failed, error: %v", resp.TypeUrl, err)
	}
}

func (ads *AdsStreamClient) Stop() {
	ads.streamClient.cancel()
	if ads.streamClient.conn != nil {
		ads.streamClient.conn.Close()
		ads.streamClient.conn = nil
	}
	ads.streamClient.client = nil
}
