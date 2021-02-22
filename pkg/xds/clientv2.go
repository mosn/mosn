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

/* Package xds can be used to create an grpc (just support grpc and v2 api) client communicated with pilot
   and fetch config in cycle
*/

package xds

import (
	"errors"
	"sync"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_cluster "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	envoy_config_bootstrap_v2 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/golang/protobuf/jsonpb"
	jsoniter "github.com/json-iterator/go"
	mv2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	v2 "mosn.io/mosn/pkg/xds/v2"
	"mosn.io/mosn/pkg/xds/v2/conv"
)

// clientv2 provide an ADS client with xDS v2
type clientv2 struct {
	config           *mv2.MOSNConfig
	dynamicResources *envoy_config_bootstrap_v2.Bootstrap_DynamicResources
	staticResources  *envoy_config_bootstrap_v2.Bootstrap_StaticResources
	adsClient        *v2.ADSClient
}

// Start used to fetch listeners/clusters/clusterloadassignment config from pilot in cycle,
// usually called when mosn start
func (c *clientv2) Start() error {
	log.DefaultLogger.Infof("xds v2 client start")

	conv.InitStats()

	xdsConfig := v2.XDSConfig{}
	err := xdsConfig.Init(c.dynamicResources, c.staticResources)
	if err != nil {
		log.DefaultLogger.Warnf("fail to init xds v2 config, skip xds: %v", err)
		return errors.New("fail to init xds v2 config")
	}

	stopChan := make(chan int)
	sendControlChan := make(chan int)
	recvControlChan := make(chan int)
	adsClient := &v2.ADSClient{
		AdsConfig:              xdsConfig.ADSConfig,
		StreamClientMutex:      sync.RWMutex{},
		StreamClient:           nil,
		MosnConfig:             c.config,
		SendControlChan:        sendControlChan,
		RecvControlChan:        recvControlChan,
		AsyncHandleControlChan: make(chan int),
		AsyncHandleChan:        make(chan *envoy_api_v2.DiscoveryResponse, 20),
		StopChan:               stopChan,
	}
	adsClient.Start()
	c.adsClient = adsClient
	return nil
}

// Stop used to stop fetch listeners/clusters/clusterloadassignment config from pilot,
// usually called when mosn quit
func (c *clientv2) Stop() {
	if c.adsClient != nil {
		log.DefaultLogger.Infof("prepare to stop xds client")
		c.adsClient.Stop()
		log.DefaultLogger.Infof("xds client stop")
	}
}

func unmarshalStaticResourcesV2(config *mv2.MOSNConfig) (staticResources *envoy_config_bootstrap_v2.Bootstrap_StaticResources, err error) {
	if len(config.RawStaticResources) < 1 {
		return nil, nil
	}

	staticResources = &envoy_config_bootstrap_v2.Bootstrap_StaticResources{}
	var b []byte
	resources := map[string]jsoniter.RawMessage{}
	err = json.Unmarshal([]byte(config.RawStaticResources), &resources)
	if err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal static_resources: %v", err)
		return nil, err
	}
	if clustersRaw, ok := resources["clusters"]; ok {
		clusters := []jsoniter.RawMessage{}
		err = json.Unmarshal([]byte(clustersRaw), &clusters)
		if err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal clusters: %v", err)
			return nil, err
		}
		for i, clusterRaw := range clusters {
			cluster := map[string]jsoniter.RawMessage{}
			err = json.Unmarshal([]byte(clusterRaw), &cluster)
			if err != nil {
				log.DefaultLogger.Errorf("fail to unmarshal cluster: %v", err)
				return nil, err
			}
			cb := envoy_api_v2_cluster.CircuitBreakers{}
			b, err = json.Marshal(&cb)
			if err != nil {
				log.DefaultLogger.Errorf("fail to marshal circuit_breakers: %v", err)
				return nil, err
			}
			cluster["circuit_breakers"] = jsoniter.RawMessage(b)

			b, err = json.Marshal(&cluster)
			if err != nil {
				log.DefaultLogger.Errorf("fail to marshal cluster: %v", err)
				return nil, err
			}
			clusters[i] = jsoniter.RawMessage(b)
		}
		b, err = json.Marshal(&clusters)
		if err != nil {
			log.DefaultLogger.Errorf("fail to marshal clusters: %v", err)
			return nil, err
		}
	}
	resources["clusters"] = jsoniter.RawMessage(b)
	b, err = json.Marshal(&resources)
	if err != nil {
		log.DefaultLogger.Errorf("fail to marshal resources: %v", err)
		return nil, err
	}

	err = jsonpb.UnmarshalString(string(b), staticResources)
	if err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal static_resources: %v", err)
		return nil, err
	}

	err = staticResources.Validate()
	if err != nil {
		log.DefaultLogger.Errorf("Invalid static_resources: %v", err)
		return nil, err
	}

	return staticResources, nil
}
