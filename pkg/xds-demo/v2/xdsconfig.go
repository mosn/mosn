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
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

//  Init parsed ds and clusters config for xds
func (c *XDSConfig) Init(dynamicResources *bootstrap.Bootstrap_DynamicResources, staticResources *bootstrap.Bootstrap_StaticResources) error {
	err := c.loadClusters(staticResources)
	if err != nil {
		return err
	}
	err = c.loadADSConfig(dynamicResources)
	if err != nil {
		return err
	}
	return nil
}

func (c *XDSConfig) loadADSConfig(dynamicResources *bootstrap.Bootstrap_DynamicResources) error {
	if dynamicResources == nil || dynamicResources.AdsConfig == nil {
		log.DefaultLogger.Errorf("DynamicResources is null")
		err := errors.New("null point exception")
		return err
	}
	err := dynamicResources.AdsConfig.Validate()
	if err != nil {
		log.DefaultLogger.Errorf("Invalid DynamicResources")
		return err
	}
	config, err := c.getAPISourceEndpoint(dynamicResources.AdsConfig)
	if err != nil {
		log.DefaultLogger.Errorf("fail to get api source endpoint")
		return err
	}
	c.ADSConfig = config
	return nil
}

func (c *XDSConfig) getAPISourceEndpoint(source *core.ApiConfigSource) (*ADSConfig, error) {
	config := &ADSConfig{}
	if source.ApiType != core.ApiConfigSource_GRPC {
		log.DefaultLogger.Errorf("unsupported api type: %v", source.ApiType)
		err := errors.New("only support GRPC api type yet")
		return nil, err
	}
	config.APIType = source.ApiType
	if source.RefreshDelay == nil || source.RefreshDelay.Nanoseconds() <= 0 {
		duration := time.Duration(time.Second * 10) // default refresh delay
		config.RefreshDelay = &duration
	} else {
		config.RefreshDelay = source.RefreshDelay
	}

	config.Services = make([]*ServiceConfig, 0, len(source.GrpcServices))
	for _, service := range source.GrpcServices {
		t := service.TargetSpecifier
		if target, ok := t.(*core.GrpcService_EnvoyGrpc_); ok {
			serviceConfig := ServiceConfig{}
			if service.Timeout == nil || (serviceConfig.Timeout.Seconds() <= 0 && serviceConfig.Timeout.Nanoseconds() <= 0) {
				duration := time.Duration(time.Second) // default connection timeout
				serviceConfig.Timeout = &duration
			} else {
				var nanos = service.Timeout.Seconds*int64(time.Second) + int64(service.Timeout.Nanos)
				duration := time.Duration(nanos)
				serviceConfig.Timeout = &duration
			}
			clusterName := target.EnvoyGrpc.ClusterName
			serviceConfig.ClusterConfig = c.Clusters[clusterName]
			if serviceConfig.ClusterConfig == nil {
				log.DefaultLogger.Errorf("cluster not found: %s", clusterName)
				return nil, fmt.Errorf("cluster not found: %s", clusterName)
			}
			config.Services = append(config.Services, &serviceConfig)
		} else if _, ok := t.(*core.GrpcService_GoogleGrpc_); ok {
			log.DefaultLogger.Warnf("GrpcService_GoogleGrpc_ not support yet")
			continue
		}
	}
	return config, nil
}

func (c *XDSConfig) loadClusters(staticResources *bootstrap.Bootstrap_StaticResources) error {
	if staticResources == nil {
		log.DefaultLogger.Errorf("StaticResources is null")
		err := errors.New("null point exception")
		return err
	}
	err := staticResources.Validate()
	if err != nil {
		log.DefaultLogger.Errorf("Invalid StaticResources")
		return err
	}
	c.Clusters = make(map[string]*ClusterConfig)
	for _, cluster := range staticResources.Clusters {
		name := cluster.Name
		config := ClusterConfig{}
		if cluster.LbPolicy != xdsapi.Cluster_RANDOM {
			log.DefaultLogger.Warnf("only random lbPoliy supported, convert to random")
		}
		config.LbPolicy = xdsapi.Cluster_RANDOM
		if cluster.ConnectTimeout.Nanoseconds() <= 0 {
			duration := time.Duration(time.Second * 10)
			config.ConnectTimeout = &duration // default connect timeout
		} else {
			config.ConnectTimeout = &cluster.ConnectTimeout
		}
		config.Address = make([]string, 0, len(cluster.Hosts))
		for _, host := range cluster.Hosts {
			if address, ok := host.Address.(*core.Address_SocketAddress); ok {
				if port, ok := address.SocketAddress.PortSpecifier.(*core.SocketAddress_PortValue); ok {
					newAddress := fmt.Sprintf("%s:%d", address.SocketAddress.Address, port.PortValue)
					config.Address = append(config.Address, newAddress)
				} else {
					log.DefaultLogger.Warnf("only PortValue supported")
					continue
				}
			} else {
				log.DefaultLogger.Warnf("only SocketAddress supported")
				continue
			}
		}
		c.Clusters[name] = &config
	}
	return nil
}

// GetEndpoint return an endpoint address by random
func (c *ClusterConfig) GetEndpoint() (string, *time.Duration) {
	if c.LbPolicy != xdsapi.Cluster_RANDOM || len(c.Address) < 1 {
		// never happen
		return "", nil
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	idx := r.Intn(len(c.Address))

	return c.Address[idx], c.ConnectTimeout
}

// GetStreamClient return a grpc stream client that connected to ads
func (c *ADSConfig) GetStreamClient() ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient {
	if c.StreamClient != nil && c.StreamClient.Client != nil {
		return c.StreamClient.Client
	}

	sc := &StreamClient{}

	if c.Services == nil {
		log.DefaultLogger.Errorf("no available ads service")
		return nil
	}
	var endpoint string
	//var timeout *time.Duration
	for _, service := range c.Services {
		if service.ClusterConfig == nil {
			continue
		}
		endpoint, _ = service.ClusterConfig.GetEndpoint()
		if len(endpoint) > 0 {
			break
		}
	}
	if len(endpoint) == 0 {
		log.DefaultLogger.Errorf("no available ads endpoint")
		return nil
	}
	conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		log.DefaultLogger.Errorf("did not connect: %v", err)
		return nil
	}
	sc.Conn = conn
	client := ads.NewAggregatedDiscoveryServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	sc.Cancel = cancel
	streamClient, err := client.StreamAggregatedResources(ctx)
	if err != nil {
		log.DefaultLogger.Errorf("fail to create stream client: %v", err)
		return nil
	}
	sc.Client = streamClient
	c.StreamClient = sc
	return streamClient
}

func (c *ADSConfig) getADSRefreshDelay() *time.Duration {
	return c.RefreshDelay
}

func (c *ADSConfig) closeADSStreamClient() {
	if c.StreamClient == nil {
		return
	}
	c.StreamClient.Cancel()
	if c.StreamClient.Conn != nil {
		c.StreamClient.Conn.Close()
		c.StreamClient.Conn = nil
	}
	c.StreamClient.Client = nil
	c.StreamClient = nil
}
