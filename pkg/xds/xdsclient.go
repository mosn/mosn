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
	"errors"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	apicluster "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
	"github.com/alipay/sofamosn/pkg/config"
	"github.com/alipay/sofamosn/pkg/log"
	"github.com/alipay/sofamosn/pkg/xds/v2"
	//"github.com/alipay/sofamosn/pkg/types"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//var warmuped chan bool = make(chan bool)
//var stopping chan bool = make(chan bool)
//var stopped chan bool = make(chan bool)
//var started bool = false

type XdsClient struct {
	v2        *v2.V2Client
	adsClient *v2.ADSClient
}

func (c *XdsClient) getConfig(config *config.MOSNConfig) error {
	log.DefaultLogger.Infof("start to get config from istio")
	err := c.getListenersAndRoutes(config)
	if err != nil {
		log.DefaultLogger.Errorf("fail to get lds config from istio")
		return err
	}
	err = c.getClustersAndHosts(config)
	if err != nil {
		log.DefaultLogger.Errorf("fail to get cds config from istio")
		return err
	}
	log.DefaultLogger.Infof("get config from istio success")
	return nil
}

func (c *XdsClient) getListenersAndRoutes(config *config.MOSNConfig) error {
	log.DefaultLogger.Infof("start to get listeners from LDS")
	streamClient := c.v2.Config.ADSConfig.GetStreamClient()
	listeners := c.v2.GetListeners(streamClient)
	if listeners == nil {
		log.DefaultLogger.Errorf("get none listeners")
		return errors.New("get none listener")
	}
	log.DefaultLogger.Infof("get %d listeners from LDS", len(listeners))
	err := config.OnUpdateListeners(listeners)
	if err != nil {
		log.DefaultLogger.Errorf("fail to update listeners")
		return errors.New("fail to update listeners")
	}
	log.DefaultLogger.Infof("update listeners success")
	return nil
}

func (c *XdsClient) getClustersAndHosts(config *config.MOSNConfig) error {
	log.DefaultLogger.Infof("start to get clusters from CDS")
	streamClient := c.v2.Config.ADSConfig.GetStreamClient()
	clusters := c.v2.GetClusters(streamClient)
	if clusters == nil {
		log.DefaultLogger.Errorf("get none clusters")
		return errors.New("get none clusters")
	}
	log.DefaultLogger.Infof("get %d clusters from CDS", len(clusters))
	err := config.OnUpdateClusters(clusters)
	if err != nil {
		log.DefaultLogger.Errorf("fall to update clusters")
		return errors.New("fail to update clusters")
	}
	log.DefaultLogger.Infof("update clusters success")

	log.DefaultLogger.Infof("start to get clusters from EDS")
	clusterNames := make([]string, 0)
	for _, cluster := range clusters {
		if cluster.Type == xdsapi.Cluster_EDS {
			clusterNames = append(clusterNames, cluster.Name)
		}
	}
	log.DefaultLogger.Debugf("start to get endpoints for cluster %v from EDS", clusterNames)
	endpoints := c.v2.GetEndpoints(streamClient, clusterNames)
	if endpoints == nil {
		log.DefaultLogger.Warnf("get none endpoints for cluster %v", clusterNames)
		return errors.New("get none endpoints for clusters")
	}
	log.DefaultLogger.Debugf("get %d endpoints for cluster %v", len(endpoints), clusterNames)
	err = config.OnUpdateEndpoints(endpoints)
	if err != nil {
		log.DefaultLogger.Errorf("fail to update endpoints for cluster %v", clusterNames)
		return errors.New("fail to update endpoints for clusters")
	}
	log.DefaultLogger.Debugf("update endpoints for cluster %v success", clusterNames)
	log.DefaultLogger.Infof("update endpoints success")
	return nil
}

func duration2String(duration *types.Duration) string {
	d := time.Duration(duration.Seconds)*time.Second + time.Duration(duration.Nanos)*time.Nanosecond
	x := fmt.Sprintf("%.9f", d.Seconds())
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, "000")
	return x + "s"
}

/*
in order to convert bootstrap_v2 json to pb struct (go-control-plane), some fields must be exchanged format
*/
func UnmarshalResources(config *config.MOSNConfig) (dynamicResources *bootstrap.Bootstrap_DynamicResources, staticResources *bootstrap.Bootstrap_StaticResources, err error) {

	if len(config.RawDynamicResources) > 0 {
		dynamicResources = &bootstrap.Bootstrap_DynamicResources{}
		resources := map[string]json.RawMessage{}
		err = json.Unmarshal([]byte(config.RawDynamicResources), &resources)
		if err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal dynamic_resources: %v", err)
			return nil, nil, err
		}
		if adsConfigRaw, ok := resources["ads_config"]; ok {
			var b []byte
			adsConfig := map[string]json.RawMessage{}
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
				adsConfig["refresh_delay"] = json.RawMessage(b)
			}
			b, err = json.Marshal(&adsConfig)
			if err != nil {
				log.DefaultLogger.Errorf("fail to marshal refresh_delay: %v", err)
				return nil, nil, err
			}
			resources["ads_config"] = json.RawMessage(b)
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
		resources := map[string]json.RawMessage{}
		err = json.Unmarshal([]byte(config.RawStaticResources), &resources)
		if err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal static_resources: %v", err)
			return nil, nil, err
		}
		if clustersRaw, ok := resources["clusters"]; ok {
			clusters := []json.RawMessage{}
			err = json.Unmarshal([]byte(clustersRaw), &clusters)
			if err != nil {
				log.DefaultLogger.Errorf("fail to unmarshal clusters: %v", err)
				return nil, nil, err
			}
			for i, clusterRaw := range clusters {
				cluster := map[string]json.RawMessage{}
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
				cluster["circuit_breakers"] = json.RawMessage(b)

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
					cluster["connect_timeout"] = json.RawMessage(b)
				}
				b, err = json.Marshal(&cluster)
				if err != nil {
					log.DefaultLogger.Errorf("fail to marshal cluster: %v", err)
					return nil, nil, err
				}
				clusters[i] = json.RawMessage(b)
			}
			b, err = json.Marshal(&clusters)
			if err != nil {
				log.DefaultLogger.Errorf("fail to marshal clusters: %v", err)
				return nil, nil, err
			}
		}
		resources["clusters"] = json.RawMessage(b)
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

func (c *XdsClient) Start(config *config.MOSNConfig, serviceCluster, serviceNode string) error {
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
		c.v2 = &v2.V2Client{serviceCluster, serviceNode, &xdsConfig}
	}
	for true {
		err := c.getConfig(config)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		break
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

func (c *XdsClient) Stop() {
	log.DefaultLogger.Infof("prepare to stop xds client")
	c.adsClient.Stop()
	log.DefaultLogger.Infof("xds client stop")
}

// must be call after func start
//func (c *XdsClient)WaitForWarmUp() {
//	<- warmuped
//}
