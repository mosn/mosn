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

func TestAddOrUpdateListener(t *testing.T) {
	// only keep useful test part
	cfg := []byte(basicConfigStr)
	mockInitConfig(t, cfg)

	lnCfg := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name: "TestAddNewListener",
		},
	}

	AddOrUpdateListener(lnCfg)

	_, idx0 := findListener("TestAddNewListener")
	if idx0 == -1 {
		t.Fatal("add new listener failed")
	}

	lnCfgNew := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name: "TestAddNewListener",
			StreamFilters: []v2.Filter{
				v2.Filter{
					Type: "test_stream_filter",
				},
			},
		},
	}

	AddOrUpdateListener(lnCfgNew)

	ln, idx1 := findListener("TestAddNewListener")
	if idx1 == -1 || idx1 != idx0 {
		t.Fatalf("update listener failed, idx0=%d, idx1=%d", idx0, idx1)
	}

	if !reflect.DeepEqual(&ln, lnCfgNew) {
		t.Fatal("config is not equal")
	}
}

func TestUpdateFullConfig(t *testing.T) {
	// only keep useful test part
	cfg := []byte(basicConfigStr)
	mockInitConfig(t, cfg)

	listeners := []v2.Listener{
		{
			ListenerConfig: v2.ListenerConfig{
				Name: "listener00",
			},
		},
		{
			ListenerConfig: v2.ListenerConfig{
				Name: "listener01",
			},
		},
	}

	routers := []*v2.RouterConfiguration{
		&v2.RouterConfiguration{
			RouterConfigurationConfig: v2.RouterConfigurationConfig{
				RouterConfigName: "router00",
			},
		},
		&v2.RouterConfiguration{
			RouterConfigurationConfig: v2.RouterConfigurationConfig{
				RouterConfigName: "router01",
			},
		},
	}

	clusters := []v2.Cluster{
		{
			Name: "cluster00",
		},
		{
			Name: "cluster01",
		},
	}

	UpdateFullConfig(listeners, routers, clusters)

	srv := config.Servers[0]
	if !(len(srv.Listeners) == 2 &&
		srv.Listeners[0].Name == "listener00" &&
		srv.Listeners[1].Name == "listener01") {
		t.Fatalf("listeners update failed")
	}

	if !(len(srv.Routers) == 2 &&
		srv.Routers[0].RouterConfigName == "router00" &&
		srv.Routers[1].RouterConfigName == "router01") {
		t.Fatalf("routers update failed")
	}

	if !(len(config.ClusterManager.Clusters) == 2 &&
		config.ClusterManager.Clusters[0].Name == "cluster00" &&
		config.ClusterManager.Clusters[1].Name == "cluster01") {
		t.Fatalf("clusters update failed")
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
			UpdateFullConfig([]v2.Listener{}, []*v2.RouterConfiguration{}, []v2.Cluster{})
		},
		func() {
			AddOrUpdateListener(&v2.Listener{})
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
