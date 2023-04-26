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
	"io/ioutil"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/featuregate"
)

// Test Feature Dump Auto
func TestFeatureDump(t *testing.T) {
	// clean the transfer function
	RegisterTransferExtension(nil)
	f := &ConfigAutoFeature{
		BaseFeatureSpec: featuregate.BaseFeatureSpec{
			DefaultValue: true,
		},
	}
	Reset()
	createMosnConfig()
	if cfg := Load(testConfigPath); cfg != nil {
		// mock mosn config parsed
		SetMosnConfig(cfg)
		for _, ln := range cfg.Servers[0].Listeners {
			SetListenerConfig(ln)
		}
		for _, c := range cfg.ClusterManager.Clusters {
			SetClusterConfig(c)
			SetHosts(c.Name, c.Hosts)
		}
		for _, r := range cfg.Servers[0].Routers {
			if r != nil {
				SetRouter(*r)
			}
		}
		for _, ext := range cfg.Extends {
			SetExtend(ext.Type, ext.Config)
		}
	}
	f.InitFunc() // start init func
	defer func() {
		enableAutoWrite = false
	}()
	// update config
	SetListenerConfig(v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       "test_mosn_listener",
			AddrConfig: "127.0.0.1:8080",
			FilterChains: []v2.FilterChain{
				{
					FilterChainConfig: v2.FilterChainConfig{
						Filters: []v2.Filter{
							{
								Type: "proxy",
								Config: map[string]interface{}{
									"downstream_protocol": "Http1",
									"upstream_protocol":   "Http1",
									"router_config_name":  "test_router",
								},
							},
						},
					},
				},
			},
			StreamFilters: []v2.Filter{
				{
					Type: "stream_filter",
				},
			},
		},
	})
	SetClusterConfig(v2.Cluster{
		Name: "cluster002",
	})
	SetHosts("cluster001", []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1:8081",
			},
		},
	})

	SetHosts("cluster002", []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1:8080",
			},
		},
	})
	SetRouter(v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: "test_router_2",
			RouterConfigPath: "/tmp/routers/test_router_2/",
		},
		VirtualHosts: []v2.VirtualHost{
			v2.VirtualHost{
				Name:    "virtualhost_1",
				Domains: []string{"*"},
			},
		},
	})
	SetRouter(v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: "test_router",
			RouterConfigPath: "/tmp/routers/test_router/",
		},
		VirtualHosts: []v2.VirtualHost{
			v2.VirtualHost{
				Name:    "virtualhost_0",
				Domains: []string{"*"},
			},
		},
	})
	SetClusterManagerTLS(v2.TLSConfig{
		Status:       true,
		InsecureSkip: true,
	})
	SetExtend("registry", json.RawMessage(`{
		"application": "test"
	}`))
	time.Sleep(4 * time.Second) // make sure feature init func trigger set dump
	DumpConfig()                // trigger dump
	// Verify Dump JSON
	content, err := DumpJSON()
	if err != nil {
		t.Fatalf("dump json error: %v", err)
	}
	// content should not contains routerConfigPath any cluster manager path
	ecfg := &effectiveConfig{}
	_ = json.Unmarshal(content, ecfg)
	if !(ecfg.MosnConfig.ClusterManager.ClusterConfigPath == "" &&
		len(ecfg.MosnConfig.ClusterManager.ClustersJson) == 0 &&
		len(ecfg.MosnConfig.ClusterManager.Clusters) == 0 &&
		ecfg.MosnConfig.ClusterManager.TLSContext.Status &&
		len(ecfg.MosnConfig.Extends) == 0 &&
		len(ecfg.routerConfigPath) == 0) {
		t.Fatalf("json dump is not expected: %+v", ecfg)
	}
	ln := ecfg.Listener["test_mosn_listener"]
	if !(ln.FilterChains[0].Filters[0].Config["downstream_protocol"] == "Http1" &&
		len(ln.StreamFilters) == 1) {
		t.Fatalf("update listener is not exepcted: %+v", ln)
	}
	if len(ecfg.Routers) != 2 {
		t.Fatal("add new router config failed")
	}
	for _, r := range ecfg.Routers {
		if r.RouterConfigPath != "" {
			t.Fatalf("%s router path is not cleaned", r.RouterConfigName)
		}
		if len(r.VirtualHosts[0].Domains) != 1 {
			t.Fatalf("%s router virtual host is not updated", r.RouterConfigName)
		}
	}
	if len(ecfg.ExtendConfigs) != 1 {
		t.Fatalf("extend config is not expected")
	}
	if len(ecfg.Cluster) != 2 {
		t.Fatal("add new cluster failed")
	}
	for _, c := range ecfg.Cluster {
		if len(c.Hosts) != 1 {
			t.Fatalf("%s hosts update failed", c.Name)
		}
	}
	// Verify path dump
	clusters, err := ioutil.ReadDir("/tmp/clusters")
	if err != nil || len(clusters) != 2 {
		t.Fatalf("cluster is not dumped into directory")
	}
	if routers, err := ioutil.ReadDir("/tmp/routers/test_router"); err != nil || len(routers) != 1 {
		t.Fatalf("router is not dumped into directory")
	}
	if routers, err := ioutil.ReadDir("/tmp/routers/test_router_2"); err != nil || len(routers) != 1 {
		t.Fatalf("router is not dumped into directory")
	}
	// read file, different than dump json
	fromFile := Load(GetConfigPath())
	if !(fromFile.Debug.StartDebug &&
		fromFile.Tracing.Enable &&
		fromFile.Metrics.StatsMatcher.RejectAll &&
		fromFile.GetAdmin().GetAddress() == "127.0.0.1") {
		t.Fatalf("basic server config is not dumped, %+v", fromFile)

	}
	if !(len(fromFile.ClusterManager.Clusters) == 2 &&
		fromFile.ClusterManager.TLSContext.Status &&
		fromFile.ClusterManager.ClusterConfigPath == "/tmp/clusters" &&
		len(fromFile.Extends) == 1) {
		t.Fatalf("basic config is not dumped, %+v", fromFile)
	}
	if ln := fromFile.Servers[0].Listeners[0]; !(ln.FilterChains[0].Filters[0].Config["downstream_protocol"] == "Http1" && len(ln.StreamFilters) == 1) {
		t.Fatalf("listener from file is not expected: %+v", ln)
	}
	if len(fromFile.Servers[0].Routers) != 2 {
		t.Fatalf("routers from file is not expected:%+v", fromFile.Servers[0].Routers)
	}
	for _, r := range fromFile.Servers[0].Routers {
		if r.RouterConfigPath == "" {
			t.Fatalf("router config path should not be null")
		}
	}
}
