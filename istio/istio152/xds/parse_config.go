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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	apicluster "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/duration"
	"mosn.io/mosn/istio/istio152/xds/conv"
	"mosn.io/mosn/pkg/admin/server"
	"mosn.io/mosn/pkg/istio"
	"mosn.io/mosn/pkg/log"
)

func init() {
	istio.RegisterParseAdsConfig(UnmarshalResources)
}

// UnmarshalResources register  istio.ParseAdsConfig
func UnmarshalResources(dynamic, static json.RawMessage) (istio.XdsStreamConfig, error) {
	ads, err := unmarshalResources(dynamic, static)
	if err != nil {
		return nil, err
	}
	// register admin api
	server.RegisterAdminHandleFunc("/stats", ads.statsForIstio)

	return ads, nil
}

// unmarshalResources used in order to convert bootstrap_v2 json to pb struct (go-control-plane), some fields must be exchanged format
func unmarshalResources(dynamic, static json.RawMessage) (*AdsConfig, error) {
	dynamicResources, err := unmarshalDynamic(dynamic)
	if err != nil {
		return nil, err
	}
	staticResources, err := unmarshalStatic(static)
	if err != nil {
		return nil, err
	}
	cfg := &AdsConfig{
		xdsInfo:   istio.GetGlobalXdsInfo(),
		converter: conv.NewConverter(),
	}
	if err := cfg.loadClusters(staticResources); err != nil {
		return nil, err
	}
	if err := cfg.loadADSConfig(dynamicResources); err != nil {
		return nil, err
	}
	return cfg, nil
}

func duration2String(duration *duration.Duration) string {
	d := time.Duration(duration.Seconds)*time.Second + time.Duration(duration.Nanos)*time.Nanosecond
	x := fmt.Sprintf("%.9f", d.Seconds())
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, "000")
	return x + "s"
}

func unmarshalDynamic(dynamic json.RawMessage) (*bootstrap.Bootstrap_DynamicResources, error) {
	// no dynamic resource, returns nil error
	if len(dynamic) <= 0 {
		return nil, nil
	}
	dynamicResources := &bootstrap.Bootstrap_DynamicResources{}
	resources := map[string]json.RawMessage{}
	if err := json.Unmarshal(dynamic, &resources); err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal dynamic_resources: %v", err)
		return nil, err
	}
	adsConfigRaw, ok := resources["ads_config"]
	if !ok {
		log.DefaultLogger.Errorf("ads_config not found")
		return nil, errors.New("lack of ads_config")
	}
	adsConfig := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(adsConfigRaw), &adsConfig); err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal ads_config: %v", err)
		return nil, err
	}
	if refreshDelayRaw, ok := adsConfig["refresh_delay"]; ok {
		refreshDelay := duration.Duration{}
		if err := json.Unmarshal([]byte(refreshDelayRaw), &refreshDelay); err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal refresh_delay: %v", err)
			return nil, err
		}

		d := duration2String(&refreshDelay)
		b, _ := json.Marshal(&d)
		adsConfig["refresh_delay"] = json.RawMessage(b)
	}
	b, err := json.Marshal(&adsConfig)
	if err != nil {
		log.DefaultLogger.Errorf("fail to marshal refresh_delay: %v", err)
		return nil, err
	}
	resources["ads_config"] = json.RawMessage(b)
	b, err = json.Marshal(&resources)
	if err != nil {
		log.DefaultLogger.Errorf("fail to marshal ads_config: %v", err)
		return nil, err
	}
	if err := jsonpb.UnmarshalString(string(b), dynamicResources); err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal dynamic_resources: %v", err)
		return nil, err
	}
	if err := dynamicResources.Validate(); err != nil {
		log.DefaultLogger.Errorf("invalid dynamic_resources: %v", err)
		return nil, err
	}
	return dynamicResources, nil
}

func unmarshalStatic(static json.RawMessage) (*bootstrap.Bootstrap_StaticResources, error) {
	if len(static) <= 0 {
		return nil, nil
	}
	staticResources := &bootstrap.Bootstrap_StaticResources{}
	resources := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(static), &resources); err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal static_resources: %v", err)
		return nil, err
	}
	var data []byte
	if clustersRaw, ok := resources["clusters"]; ok {
		clusters := []json.RawMessage{}
		if err := json.Unmarshal([]byte(clustersRaw), &clusters); err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal clusters: %v", err)
			return nil, err
		}
		for i, clusterRaw := range clusters {
			cluster := map[string]json.RawMessage{}
			if err := json.Unmarshal([]byte(clusterRaw), &cluster); err != nil {
				log.DefaultLogger.Errorf("fail to unmarshal cluster: %v", err)
				return nil, err
			}
			cb := apicluster.CircuitBreakers{}
			b, err := json.Marshal(&cb)
			if err != nil {
				log.DefaultLogger.Errorf("fail to marshal circuit_breakers: %v", err)
				return nil, err
			}
			cluster["circuit_breakers"] = json.RawMessage(b)
			b, err = json.Marshal(&cluster)
			if err != nil {
				log.DefaultLogger.Errorf("fail to marshal cluster: %v", err)
				return nil, err
			}
			clusters[i] = json.RawMessage(b)
		}
		b, err := json.Marshal(&clusters)
		if err != nil {
			log.DefaultLogger.Errorf("fail to marshal clusters: %v", err)
			return nil, err
		}
		data = b
	}
	resources["clusters"] = json.RawMessage(data)
	b, err := json.Marshal(&resources)
	if err != nil {
		log.DefaultLogger.Errorf("fail to marshal resources: %v", err)
		return nil, err
	}
	if err := jsonpb.UnmarshalString(string(b), staticResources); err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal static_resources: %v", err)
		return nil, err
	}
	if err := staticResources.Validate(); err != nil {
		log.DefaultLogger.Errorf("Invalid static_resources: %v", err)
		return nil, err
	}
	return staticResources, nil
}
