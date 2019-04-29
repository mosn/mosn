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

package config

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

func mockInitConfig(t *testing.T, cfg []byte) {
	// config is a global var
	if err := json.Unmarshal(cfg, &config); err != nil {
		t.Fatal("init config failed", err)
	}
}

func TestUpdateClusterConfig(t *testing.T) {
	// only keep useful test part
	cfg := []byte(basicClusterConfigStr)
	mockInitConfig(t, cfg)
	// add a cluster
	clusterConfigStr := `{
		"name": "test_new",
		"type": "SIMPLE",
		"lb_type": "LB_RANDOM",
		"hosts": []
	}`
	clusterNew := v2.Cluster{}
	if err := json.Unmarshal([]byte(clusterConfigStr), &clusterNew); err != nil {
		t.Fatal("unmarshal cluster config err", err)
	}
	addOrUpdateClusterConfig([]v2.Cluster{clusterNew})
	// verify
	if len(config.ClusterManager.Clusters) != 2 {
		t.Fatal("add cluster failed")
	}
	// update cluster
	clusterNew.ClusterType = v2.EDS_CLUSTER
	addOrUpdateClusterConfig([]v2.Cluster{clusterNew})
	// verify
	for _, c := range config.ClusterManager.Clusters {
		if c.Name == "test_new" {
			if c.ClusterType != v2.EDS_CLUSTER {
				t.Error("update cluster failed")
			}
		}
	}
	// remove clustere
	if !removeClusterConfig([]string{"test_new"}) {
		t.Fatal("remove test new cluster failed")
	}
	// verify
	if len(config.ClusterManager.Clusters) != 1 {
		t.Fatal("remove cluster failed")
	}
}

func TestUpdateRouterConfig(t *testing.T) {
	// only keep useful test part
	cfg := []byte(basicConfigStr)
	mockInitConfig(t, cfg)
	routerConfigStr := `{
		"router_config_name":"test_update",
		"virtual_hosts":[{
			"name":"test_update",
			"domains": ["*"],
			"routers": [
				{
					 "match": {"prefix":"/test_update"},
					 "route":{"cluster_name":"test_update"}
				}
			]
		}]
	}`
	routerConfiguration := &v2.RouterConfiguration{}
	if err := json.Unmarshal([]byte(routerConfigStr), routerConfiguration); err != nil {
		t.Fatal("create update config failed", err)
	}
	if !addOrUpdateRouterConfig("egress", routerConfiguration) {
		t.Fatal("update router config failed")
	}
	dumpRouterConfig()
	// verify
	ln, idx := findListener("egress")
	if idx == -1 {
		t.Fatal("cannot found egress listener")
	}
	filter := ln.FilterChains[0].Filters[0] // only one connection_manager
	newConfig := &v2.RouterConfiguration{}
	if data, err := json.Marshal(filter.Config); err == nil {
		if err := json.Unmarshal(data, &newConfig); err != nil {
			t.Error("invalid config in router config", err)
		}
	}
	if !reflect.DeepEqual(newConfig, routerConfiguration) {
		t.Error("new config is not equal update config")
	}
}

func TestUpdateStreamFilter(t *testing.T) {
	// only keep useful test part
	cfg := []byte(basicConfigStr)
	mockInitConfig(t, cfg)
	streamFilterStr := `{
		"version": "2.0"
	}`
	streamFilterConfig := make(map[string]interface{})
	if err := json.Unmarshal([]byte(streamFilterStr), &streamFilterConfig); err != nil {
		t.Fatal("create filter config failed", err)
	}
	if !addOrUpdateStreamFilters("egress", "test", streamFilterConfig) {
		t.Fatal("update stream filter config failed")
	}
	if !addOrUpdateStreamFilters("ingress", "test", streamFilterConfig) {
		t.Fatal("add stream filter config failed")
	}
	// verify
	for _, name := range []string{"egress", "ingress"} {
		ln, idx := findListener(name)
		if idx == -1 {
			t.Fatalf("%s cannot found egress listener", name)
		}
		filter := ln.StreamFilters[0] // only one stream filter
		newConfig := filter.Config
		v, ok := newConfig["version"]
		if !ok {
			t.Fatalf("%s no version config", name)
		}
		ver := v.(string)
		if ver != "2.0" {
			t.Error("%s stream filter config update not expected", name)
		}
	}
}
