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

package configmanager

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
)

func mockInitConfig(t *testing.T, cfg []byte) {
	// config is a global var
	if err := json.Unmarshal(cfg, &config); err != nil {
		t.Fatal("init config failed", err)
	}
}

func TestUpdateClusterHost(t *testing.T) {
	// only keep useful test part
	cfg := []byte(basicConfigStr)
	mockInitConfig(t, cfg)
	clusterName := "test_cluster"
	hostCfg := v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1:8080",
		},
	}
	AddOrUpdateClusterHost(clusterName, hostCfg)
	// verify
	{
		c := config.ClusterManager.Clusters[0]
		if len(c.Hosts) != 1 {
			t.Fatal("cluster host added failed")
		}
		if c.Hosts[0].Weight != 0 {
			t.Fatal("unexpected host info")
		}
	}
	AddOrUpdateClusterHost(clusterName, v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1:8080",
			Weight:  100,
		},
	})
	{
		c := config.ClusterManager.Clusters[0]
		if len(c.Hosts) != 1 {
			t.Fatal("cluster host update failed")
		}
		if c.Hosts[0].Weight != 100 {
			t.Fatal("unexpected host info")
		}
	}
	DeleteClusterHost(clusterName, "127.0.0.1:8080")
	{
		c := config.ClusterManager.Clusters[0]
		if len(c.Hosts) != 0 {
			t.Fatal("cluster host delete failed")
		}
	}
}

func TestUpdateClusterConfig(t *testing.T) {
	// only keep useful test part
	cfg := []byte(basicConfigStr)
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
		"router_config_name":"egress_router",
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
	if !addOrUpdateRouterConfig(routerConfiguration) {
		t.Fatal("update router config failed")
	}
	// verify
	routers := config.Servers[0].Routers
	if len(routers) != 2 {
		t.Fatal("router update failed")
	}
	newConfig := &v2.RouterConfiguration{}
	for _, r := range routers {
		if r.RouterConfigName == "egress_router" {
			newConfig = r
		}
	}
	if !reflect.DeepEqual(newConfig, routerConfiguration) {
		t.Error("new config is not equal update config")
	}
}

func TestAddRouterConfig(t *testing.T) {
	// only keep useful test part
	cfg := []byte(basicConfigStr)
	mockInitConfig(t, cfg)
	routerConfigStr := `{
		"router_config_name":"test_add",
		"virtual_hosts":[{
			"name":"test_add",
			"domains": ["*"],
			"routers": [
				{
					 "match": {"prefix":"/test_add"},
					 "route":{"cluster_name":"test_add"}
				}
			]
		}]
	}`
	routerConfiguration := &v2.RouterConfiguration{}
	if err := json.Unmarshal([]byte(routerConfigStr), routerConfiguration); err != nil {
		t.Fatal("create update config failed", err)
	}
	if !addOrUpdateRouterConfig(routerConfiguration) {
		t.Fatal("update router config failed")
	}
	// verify
	routers := config.Servers[0].Routers
	if len(routers) != 3 {
		t.Fatal("router add failed")
	}
	newConfig := &v2.RouterConfiguration{}
	for _, r := range routers {
		if r.RouterConfigName == "test_add" {
			newConfig = r
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
			t.Errorf("%s stream filter config update not expected", name)
		}
	}
}

func TestUpdateMqClientKey(t *testing.T) {
	UpdateMqClientKey("hello", "ck", false)
	if len(config.ServiceRegistry.MqClientKey) != 1 {
		t.Errorf("len(config.ServiceRegistry.MqClientKey) != 1")
	}

	UpdateMqClientKey("hello", "", true)
	if len(config.ServiceRegistry.MqClientKey) != 0 {
		t.Errorf("len(config.ServiceRegistry.MqClientKey) != 0")
	}
}

func TestUpdateMqMeta(t *testing.T) {
	UpdateMqMeta("TP_TEST", "meta", false)
	if len(config.ServiceRegistry.MqMeta) != 1 {
		t.Errorf("len(config.ServiceRegistry.MqMeta) != 1")
	}

	UpdateMqMeta("TP_TEST", "meta", true)
	if len(config.ServiceRegistry.MqMeta) != 0 {
		t.Errorf("len(config.ServiceRegistry.MqMeta) != 0")
	}
}

func TestSetMqConsumers(t *testing.T) {
	SetMqConsumers("TP_TEST", []string{"cs1", "cs2", "cs3"})
	if len(config.ServiceRegistry.MqConsumers) != 1 {
		t.Errorf("len(config.ServiceRegistry.MqConsumers) != 1")
	}

	SetMqConsumers("TP_TEST", []string{})
	if len(config.ServiceRegistry.MqConsumers) != 0 {
		t.Errorf("len(config.ServiceRegistry.MqConsumers) != 0")
	}
}

func TestRmMqConsumers(t *testing.T) {
	SetMqConsumers("TP_TEST", []string{"cs1", "cs2", "cs3"})
	RmMqConsumers("TP_TEST")
	if len(config.ServiceRegistry.MqConsumers) != 0 {
		t.Errorf("len(config.ServiceRegistry.MqConsumers) != 0")
	}
}

// test avoid dead lock
func TestUpdateConfigConcurrency(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	cfg := []byte(basicConfigStr)
	mockInitConfig(t, cfg)
	wg := sync.WaitGroup{}
	for _, fc := range []func(){
		func() {
			ResetServiceRegistryInfo(v2.ApplicationInfo{}, []string{})
		},
		func() {
			AddOrUpdateClusterConfig([]v2.Cluster{})
		},
		func() {
			RemoveClusterConfig([]string{})
		},
		func() {
			AddPubInfo(map[string]string{
				"key": "value",
			})
		},
		func() {
			DelPubInfo("key")
		},
		func() {
			AddClusterWithRouter([]v2.Cluster{}, &v2.RouterConfiguration{})
		},
		func() {
			AddOrUpdateRouterConfig(&v2.RouterConfiguration{})
		},
		func() {
			AddOrUpdateStreamFilters("egress", "test", map[string]interface{}{})
		},
		func() {
			AddMsgMeta("data", "group")
		},
		func() {
			DelMsgMeta("data")
		},
		func() {
			UpdateMqClientKey("id", "key", false)
			UpdateMqClientKey("id", "key", true)
		},
		func() {
			UpdateMqMeta("topic", "meta", false)
			UpdateMqMeta("topic", "meta", true)
		},
		func() {
			SetMqConsumers("key", []string{})
		},
		func() {
			RmMqConsumers("key")
		},
	} {
		f := fc
		wg.Add(1)
		go func() {
			for i := 0; i < 1000; i++ {
				ri := rand.Intn(3000)
				f()
				time.Sleep(time.Duration(ri))
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
