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
	"fmt"
	"strings"
	"time"

	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/xds/v2"
	apicluster "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
	"github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Client provide an ADS client
type Client struct {
	v2        *v2.ClientV2
	adsClient *v2.ADSClient
}

func duration2String(duration *types.Duration) string {
	d := time.Duration(duration.Seconds)*time.Second + time.Duration(duration.Nanos)*time.Nanosecond
	x := fmt.Sprintf("%.9f", d.Seconds())
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, "000")
	return x + "s"
}

// UnmarshalResources used in order to convert bootstrap_v2 json to pb struct (go-control-plane), some fields must be exchanged format
func UnmarshalResources(config *config.MOSNConfig) (dynamicResources *bootstrap.Bootstrap_DynamicResources, staticResources *bootstrap.Bootstrap_StaticResources, err error) {

	if len(config.RawDynamicResources) > 0 {
		dynamicResources = &bootstrap.Bootstrap_DynamicResources{}
		resources := map[string]jsoniter.RawMessage{}
		err = json.Unmarshal([]byte(config.RawDynamicResources), &resources)
		if err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal dynamic_resources: %v", err)
			return nil, nil, err
		}
		if adsConfigRaw, ok := resources["ads_config"]; ok {
			var b []byte
			adsConfig := map[string]jsoniter.RawMessage{}
			err = json.Unmarshal([]byte(adsConfigRaw), &adsConfig)
			if err != nil {
				log.DefaultLogger.Errorf("fail to unmarshal ads_config: %v", err)
				return nil, nil, err
			}
			if refreshDelayRaw, ok := adsConfig["refresh_delay"]; ok {
				refreshDelay := types.Duration{}
				err = json.Unmarshal([]byte(refreshDelayRaw), &refreshDelay)
				if err != nil {
					log.DefaultLogger.Errorf("fail to unmarshal refresh_delay: %v", err)
					return nil, nil, err
				}

				d := duration2String(&refreshDelay)
				b, err = json.Marshal(&d)
				adsConfig["refresh_delay"] = jsoniter.RawMessage(b)
			}
			b, err = json.Marshal(&adsConfig)
			if err != nil {
				log.DefaultLogger.Errorf("fail to marshal refresh_delay: %v", err)
				return nil, nil, err
			}
			resources["ads_config"] = jsoniter.RawMessage(b)
			b, err = json.Marshal(&resources)
			if err != nil {
				log.DefaultLogger.Errorf("fail to marshal ads_config: %v", err)
				return nil, nil, err
			}

			err = jsonpb.UnmarshalString(string(b), dynamicResources)
			if err != nil {
				log.DefaultLogger.Errorf("fail to marshal dynamic_resources: %v", err)
				return nil, nil, err
			}
			err = dynamicResources.Validate()
			if err != nil {
				log.DefaultLogger.Errorf("invalid dynamic_resources: %v", err)
				return nil, nil, err
			}
		} else {
			log.DefaultLogger.Errorf("ads_config not found")
			err = errors.New("lack of ads_config")
			return nil, nil, err
		}
	}

	if len(config.RawStaticResources) > 0 {
		staticResources = &bootstrap.Bootstrap_StaticResources{}
		var b []byte
		resources := map[string]jsoniter.RawMessage{}
		err = json.Unmarshal([]byte(config.RawStaticResources), &resources)
		if err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal static_resources: %v", err)
			return nil, nil, err
		}
		if clustersRaw, ok := resources["clusters"]; ok {
			clusters := []jsoniter.RawMessage{}
			err = json.Unmarshal([]byte(clustersRaw), &clusters)
			if err != nil {
				log.DefaultLogger.Errorf("fail to unmarshal clusters: %v", err)
				return nil, nil, err
			}
			for i, clusterRaw := range clusters {
				cluster := map[string]jsoniter.RawMessage{}
				err = json.Unmarshal([]byte(clusterRaw), &cluster)
				if err != nil {
					log.DefaultLogger.Errorf("fail to unmarshal cluster: %v", err)
					return nil, nil, err
				}
				cb := apicluster.CircuitBreakers{}
				b, err = json.Marshal(&cb)
				if err != nil {
					log.DefaultLogger.Errorf("fail to marshal circuit_breakers: %v", err)
					return nil, nil, err
				}
				cluster["circuit_breakers"] = jsoniter.RawMessage(b)

				if connectTimeoutRaw, ok := cluster["connect_timeout"]; ok {
					connectTimeout := types.Duration{}
					err = json.Unmarshal([]byte(connectTimeoutRaw), &connectTimeout)
					if err != nil {
						log.DefaultLogger.Errorf("fail to unmarshal connect_timeout: %v", err)
						return nil, nil, err
					}
					d := duration2String(&connectTimeout)
					b, err = json.Marshal(&d)
					if err != nil {
						log.DefaultLogger.Errorf("fail to marshal connect_timeout: %v", err)
						return nil, nil, err
					}
					cluster["connect_timeout"] = jsoniter.RawMessage(b)
				}
				b, err = json.Marshal(&cluster)
				if err != nil {
					log.DefaultLogger.Errorf("fail to marshal cluster: %v", err)
					return nil, nil, err
				}
				clusters[i] = jsoniter.RawMessage(b)
			}
			b, err = json.Marshal(&clusters)
			if err != nil {
				log.DefaultLogger.Errorf("fail to marshal clusters: %v", err)
				return nil, nil, err
			}
		}
		resources["clusters"] = jsoniter.RawMessage(b)
		b, err = json.Marshal(&resources)
		if err != nil {
			log.DefaultLogger.Errorf("fail to marshal resources: %v", err)
			return nil, nil, err
		}

		err = jsonpb.UnmarshalString(string(b), staticResources)
		if err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal static_resources: %v", err)
			return nil, nil, err
		}

		err = staticResources.Validate()
		if err != nil {
			log.DefaultLogger.Errorf("Invalid static_resources: %v", err)
			return nil, nil, err
		}
	}
	return dynamicResources, staticResources, nil
}

// Start used to fetch listeners/clusters/clusterloadassignment config from pilot in cycle,
// usually called when mosn start
func (c *Client) Start(config *config.MOSNConfig, serviceCluster, serviceNode string) error {
	log.DefaultLogger.Infof("xds client start")
	if c.v2 == nil {
		dynamicResources, staticResources, err := UnmarshalResources(config)
		if err != nil {
			log.DefaultLogger.Warnf("fail to unmarshal xds resources, skip xds: %v", err)
			return errors.New("fail to unmarshal xds resources")
		}
		xdsConfig := v2.XDSConfig{}
		err = xdsConfig.Init(dynamicResources, staticResources)
		if err != nil {
			log.DefaultLogger.Warnf("fail to init xds config, skip xds: %v", err)
			return errors.New("fail to init xds config")
		}
		c.v2 = &v2.ClientV2{
			ServiceCluster: serviceCluster,
			ServiceNode:    serviceNode,
			Config:         &xdsConfig,
		}
	}

	stopChan := make(chan int)
	sendControlChan := make(chan int)
	recvControlChan := make(chan int)
	adsClient := &v2.ADSClient{
		AdsConfig:       c.v2.Config.ADSConfig,
		StreamClient:    nil,
		V2Client:        c.v2,
		MosnConfig:      nil,
		SendControlChan: sendControlChan,
		RecvControlChan: recvControlChan,
		StopChan:        stopChan,
	}
	adsClient.Start()
	c.adsClient = adsClient
	return nil
}

// Stop used to stop fetch listeners/clusters/clusterloadassignment config from pilot,
// usually called when mosn quit
func (c *Client) Stop() {
	log.DefaultLogger.Infof("prepare to stop xds client")
	c.adsClient.Stop()
	log.DefaultLogger.Infof("xds client stop")
}
