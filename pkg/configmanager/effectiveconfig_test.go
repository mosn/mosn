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
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
)

type checkFunction func() error

func setupSubTest(t *testing.T) func(t *testing.T) {
	return func(t *testing.T) {
		Reset()
	}
}

func TestSetMosnConfig(t *testing.T) {
	Reset()
	createMosnConfig()
	// Set MosnConfig
	rawcfg := &v2.MOSNConfig{}
	if err := json.Unmarshal([]byte(mosnConfig), rawcfg); err != nil {
		t.Fatal(err)
	}
	SetMosnConfig(rawcfg)
	// the parameters should not be changed
	if !(rawcfg.ClusterManager.ClusterConfigPath != "" &&
		len(rawcfg.Servers[0].Listeners) > 0 &&
		len(rawcfg.Servers[0].Routers) > 0) {
		t.Fatalf("the parameter config has been changed: %+v", rawcfg)
	}
	// the stored config contains no dynamic config, such as listeners and clustersy
	if !(reflect.DeepEqual(conf.MosnConfig.ClusterManager, v2.ClusterManagerConfig{
		ClusterManagerConfigJson: v2.ClusterManagerConfigJson{
			TLSContext: rawcfg.ClusterManager.TLSContext,
		},
	}) &&
		conf.clusterConfigPath == "/tmp/clusters" &&
		len(conf.MosnConfig.Servers[0].Listeners) == 0 &&
		len(conf.MosnConfig.Servers[0].Routers) == 0) {
		t.Fatalf("the stored config contains dynamic configs: %+v", conf.MosnConfig)
	}

	SetClusterManagerTLS(v2.TLSConfig{
		Status:       true,
		InsecureSkip: true,
	})

	HandleMOSNConfig(CfgTypeMOSN, func(v interface{}) {
		cfg := v.(v2.MOSNConfig)
		if !cfg.Metrics.StatsMatcher.RejectAll {
			t.Fatalf("mosn config is not exepcted %+v", v)
		}
		ctx := cfg.ClusterManager.TLSContext
		if !(ctx.Status && ctx.InsecureSkip) {
			t.Fatalf("mosn tls manager is not expected: %+v", v)
		}
	})
}

func TestSetListenerConfig(t *testing.T) {
	cases := []struct {
		name      string
		listeners []v2.Listener
		expect    checkFunction
	}{
		{
			name: "add_new_listener",
			listeners: []v2.Listener{
				{
					ListenerConfig: v2.ListenerConfig{
						Name:       "test",
						BindToPort: false,
					},
				},
			},
			expect: func() error {
				v := getMOSNConfig(CfgTypeListener)
				lns := v.(map[string]v2.Listener)
				if len(lns) != 1 && lns["test"].Name != "test" {
					return errors.New("listener add failed")
				}
				if len(lns["test"].FilterChains) != 0 {
					return errors.New("listener add failed, FilterChains should be empty")
				}
				return nil
			},
		},
		{
			name: "update_listener",
			listeners: []v2.Listener{
				// add a new listener
				{
					ListenerConfig: v2.ListenerConfig{
						Name:       "test",
						BindToPort: false,
					},
				},
				// update listener
				{
					ListenerConfig: v2.ListenerConfig{
						Name:       "test",
						BindToPort: false,
						FilterChains: []v2.FilterChain{
							{
								FilterChainConfig: v2.FilterChainConfig{
									Filters: []v2.Filter{
										{
											Type: "xxx",
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
				},
			},
			expect: func() error {
				v := getMOSNConfig(CfgTypeListener)
				lns := v.(map[string]v2.Listener)
				if len(lns) != 1 && lns["test"].Name != "test" {
					return errors.New("listener add failed")
				}
				if len(lns["test"].FilterChains) != 1 {
					return errors.New("listener update failed, FilterChains count should be one")
				}
				if len(lns["test"].StreamFilters) != 1 {
					return errors.New("listener update failed, stream filters count should be one")
				}
				return nil

			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tearDownSubTest := setupSubTest(t)
			defer tearDownSubTest(t)
			for _, listener := range tc.listeners {
				SetListenerConfig(listener)
			}
			if err := tc.expect(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestSetClusterAndHosts(t *testing.T) {
	cases := []struct {
		name     string
		clusters []v2.Cluster
		hosts    map[string][]v2.Host
		expect   checkFunction
	}{
		{
			name: "Add Cluster & Hosts",
			clusters: []v2.Cluster{
				{
					Name:        "outbound|9080||productpage.default.svc.cluster.local",
					ClusterType: "EDS",
					SubType:     "",
					LbType:      "LB_ROUNDROBIN",
					Hosts:       nil,
				},
			},
			hosts: map[string][]v2.Host{
				"outbound|9080||productpage.default.svc.cluster.local": {
					{
						HostConfig: v2.HostConfig{
							Address: "172.16.1.154:9080",
							Weight:  1,
							MetaDataConfig: &v2.MetadataConfig{
								MetaKey: v2.LbMeta{
									LbMetaKey: nil,
								},
							},
						},
					},
				},
			},
			expect: func() error {
				v := getMOSNConfig(CfgTypeCluster)
				cs := v.(map[string]v2.Cluster)
				if len(cs) != 1 || len(cs["outbound|9080||productpage.default.svc.cluster.local"].Hosts) != 1 {
					return errors.New("[outbound|9080||productpage.default.svc.cluster.local] not exists or empty")
				}
				hosts := cs["outbound|9080||productpage.default.svc.cluster.local"].Hosts
				if hosts[0].Address != "172.16.1.154:9080" {
					return errors.New("[outbound|9080||productpage.default.svc.cluster.local] host address should be 172.16.1.154:9080, but got " + hosts[0].Address)
				}
				return nil
			},
		},
		{
			name: "Update Cluster",
			clusters: []v2.Cluster{
				{
					Name:        "outbound|9080||productpage.default.svc.cluster.local",
					ClusterType: "EDS",
					SubType:     "",
					LbType:      "LB_ROUNDROBIN",
					Hosts:       nil,
				},
				{
					Name:        "outbound|9080||productpage.default.svc.cluster.local",
					ClusterType: "EDS",
					SubType:     "",
					LbType:      "LB_RANDOM",
					Hosts:       nil,
				},
			},
			hosts: map[string][]v2.Host{},
			expect: func() error {
				v := getMOSNConfig(CfgTypeCluster)
				cs := v.(map[string]v2.Cluster)
				if len(cs) != 1 {
					return errors.New("[outbound|9080||productpage.default.svc.cluster.local] not exists or empty")
				}
				lbType := cs["outbound|9080||productpage.default.svc.cluster.local"].LbType
				if lbType != v2.LB_RANDOM {
					return errors.New(fmt.Sprintf("[outbound|9080||productpage.default.svc.cluster.local] lbType should be LB_RANDOM, but got %s", lbType))
				}
				return nil

			},
		},
		{
			name: "Update Hosts",
			clusters: []v2.Cluster{
				{
					Name:        "outbound|9080||productpage.default.svc.cluster.local",
					ClusterType: "EDS",
					SubType:     "",
					LbType:      "LB_ROUNDROBIN",
					Hosts: []v2.Host{
						{
							HostConfig: v2.HostConfig{
								Address: "172.16.1.154:9080",
								Weight:  1,
								MetaDataConfig: &v2.MetadataConfig{
									MetaKey: v2.LbMeta{
										LbMetaKey: nil,
									},
								},
							},
						},
					},
				},
			},
			hosts: map[string][]v2.Host{
				"outbound|9080||productpage.default.svc.cluster.local": {
					{
						HostConfig: v2.HostConfig{
							Address: "172.16.1.154:9080",
							Weight:  1,
							MetaDataConfig: &v2.MetadataConfig{
								MetaKey: v2.LbMeta{
									LbMetaKey: nil,
								},
							},
						},
					},
					{
						HostConfig: v2.HostConfig{
							Address: "172.16.1.155:9080",
							Weight:  3,
							MetaDataConfig: &v2.MetadataConfig{
								MetaKey: v2.LbMeta{
									LbMetaKey: nil,
								},
							},
						},
					},
				},
			},
			expect: func() error {
				v := getMOSNConfig(CfgTypeCluster)
				cs := v.(map[string]v2.Cluster)
				if len(cs) != 1 || len(cs["outbound|9080||productpage.default.svc.cluster.local"].Hosts) != 2 {
					return errors.New("[outbound|9080||productpage.default.svc.cluster.local] not exists or hosts number error")
				}
				hosts := cs["outbound|9080||productpage.default.svc.cluster.local"].Hosts
				if hosts[0].Address != "172.16.1.154:9080" {
					return errors.New("[outbound|9080||productpage.default.svc.cluster.local] hosts[0] address should be 172.16.1.154:9080, but got " + hosts[0].Address)
				}
				if hosts[1].Address != "172.16.1.155:9080" {
					return errors.New("[outbound|9080||productpage.default.svc.cluster.local] hosts[1] address should be 172.16.1.155:9080, but got " + hosts[1].Address)
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tearDownSubTest := setupSubTest(t)
			defer tearDownSubTest(t)

			for _, cluster := range tc.clusters {
				SetClusterConfig(cluster)
			}

			for clusterName, host := range tc.hosts {
				SetHosts(clusterName, host)
			}

			if err := tc.expect(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestSetRouter(t *testing.T) {
	cases := []struct {
		name    string
		routers []v2.RouterConfiguration
		expect  checkFunction
	}{
		{
			name: "add new router config",
			routers: []v2.RouterConfiguration{
				func() v2.RouterConfiguration {
					s := `{
						"router_config_name": "new_router_name",
						"virtual_hosts":[{
							"name": "virtual_host",
							"domains": ["*"],
							"routers": [
								{
									"match": {"prefix":"/test_add"},
									"route":{"cluster_name":"route_cluster"}
								}
							]
						}]
					}`
					r := v2.RouterConfiguration{}
					if err := json.Unmarshal([]byte(s), &r); err != nil {
						t.Fatalf("unmarshal router error: %v", err)
					}
					return r
				}(),
			},
			expect: func() error {
				v := getMOSNConfig(CfgTypeRouter)
				routers := v.(map[string]v2.RouterConfiguration)
				router, ok := routers["new_router_name"]
				if !ok {
					return errors.New("no router found")
				}
				if len(router.VirtualHosts) != 1 {
					return errors.New("router virtual host config is invalid")
				}
				if len(router.VirtualHosts[0].Routers) != 1 {
					return errors.New("router route rules is invalid")
				}
				return nil
			},
		},
		{
			name: "add new path mode router",
			routers: []v2.RouterConfiguration{
				func() v2.RouterConfiguration {
					s := `{
						"router_config_name": "new_path_router",
						"router_configs": "/tmp/new_path_router"
					}`
					os.RemoveAll("/tmp/new_path_router")
					os.MkdirAll("/tmp/new_path_router", 0755)
					ioutil.WriteFile("/tmp/new_path_router/virtual_host.json", []byte(`{
						"name": "virtual_host",
						"domains": ["*"],
						"routers": [
							{
								"match": {"prefix":"/"},
								"route":{"cluster_name":"route_cluster"}
							}
						]
					}`), 0644)
					r := v2.RouterConfiguration{}
					if err := json.Unmarshal([]byte(s), &r); err != nil {
						t.Fatalf("unmarshal router error: %v", err)
					}
					return r
				}(),
			},
			expect: func() error {
				v := getMOSNConfig(CfgTypeRouter)
				routers := v.(map[string]v2.RouterConfiguration)
				router, ok := routers["new_path_router"]
				if !ok {
					return errors.New("no router found")
				}
				if len(router.VirtualHosts) != 1 {
					return errors.New("router virtual host config is invalid")
				}
				if len(router.VirtualHosts[0].Routers) != 1 {
					return errors.New("router route rules is invalid")
				}
				if router.RouterConfigPath != "" {
					return errors.New("route path should be cleand")
				}
				if p, ok := conf.routerConfigPath["new_path_router"]; !ok || p != "/tmp/new_path_router" {
					return errors.New("route path should be stored")
				}
				return nil

			},
		},
		{
			name: "update route rule in router",
			routers: []v2.RouterConfiguration{
				// new route rule
				func() v2.RouterConfiguration {
					s := `{
						"router_config_name": "update_route_rule",
						"router_configs": "/tmp/update_route_rule"
					}`
					os.RemoveAll("/tmp/update_route_rule")
					os.MkdirAll("/tmp/update_route_rule", 0755)
					ioutil.WriteFile("/tmp/update_route_rule/virtual_host.json", []byte(`{
						"name": "virtual_host",
						"domains": ["*"],
						"routers": []
					}`), 0644)
					r := v2.RouterConfiguration{}
					if err := json.Unmarshal([]byte(s), &r); err != nil {
						t.Fatalf("unmarshal router error: %v", err)
					}
					return r

				}(),
				// update route rule
				func() v2.RouterConfiguration {
					s := `{
						"router_config_name": "update_route_rule",
						"router_configs": "/tmp/update_route_rule"
					}`
					ioutil.WriteFile("/tmp/update_route_rule/virtual_host.json", []byte(`{
						"name": "virtual_host",
						"domains": ["*"],
						"routers": [
							{
								"match": {"prefix":"/"},
								"route":{"cluster_name":"route_cluster"}
							}
						]
					}`), 0644)
					ioutil.WriteFile("/tmp/update_route_rule/virtual_host_add.json", []byte(`{
						 "name": "virtual_host_add",
						 "domains": ["*"],
						 "routers": []
					}`), 0644)
					r := v2.RouterConfiguration{}
					if err := json.Unmarshal([]byte(s), &r); err != nil {
						t.Fatalf("unmarshal router error: %v", err)
					}
					return r
				}(),
			},
			expect: func() error {
				v := getMOSNConfig(CfgTypeRouter)
				routers := v.(map[string]v2.RouterConfiguration)
				router, ok := routers["update_route_rule"]
				if !ok {
					return errors.New("no router found")
				}
				if len(router.VirtualHosts) != 2 {
					return errors.New("router virtual host config is invalid")
				}
				for _, vh := range router.VirtualHosts {
					switch vh.Name {
					case "virtual_host":
						if len(vh.Routers) != 1 {
							return errors.New("udpate route rule failed")
						}
					case "virtual_host_add":
						if len(vh.Routers) != 0 {
							return errors.New("udpate route rule failed")
						}
					}
				}
				if router.RouterConfigPath != "" {
					return errors.New("route path should be cleand")
				}
				if p, ok := conf.routerConfigPath["update_route_rule"]; !ok || p != "/tmp/update_route_rule" {
					return errors.New("route path should be stored")
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tearDownSubTest := setupSubTest(t)
			defer tearDownSubTest(t)

			for _, r := range tc.routers {
				SetRouter(r)
			}

			if err := tc.expect(); err != nil {
				t.Error(err)

			}
		})
	}
}

func TestRemoveClusterConfig(t *testing.T) {
	cc := []v2.Cluster{
		{
			Name:        "outbound|9080||productpage.default.svc.cluster.local",
			ClusterType: "EDS",
			SubType:     "",
			LbType:      "LB_ROUNDROBIN",
			Hosts:       nil,
		},
		{
			Name:        "test",
			ClusterType: "EDS",
			LbType:      "LB_ROUNDROBIN",
		},
	}
	for _, c := range cc {
		SetClusterConfig(c)
	}
	SetRemoveClusterConfig(cc[0].Name)
	HandleMOSNConfig(CfgTypeCluster, func(v interface{}) {
		cs := v.(map[string]v2.Cluster)
		if len(cs) != 1 {
			t.Fatal("test remove cluster failed")
		}
		if _, ok := cs["test"]; !ok {
			t.Fatal("test remove cluster failed")
		}
	})
}

func TestSetExtend(t *testing.T) {
	type extConfig struct {
		ExtInt   int
		ExtStr   string
		ExtBool  bool
		ExtSlice []int
	}
	ext := &extConfig{
		ExtInt:   10,
		ExtStr:   "test",
		ExtBool:  true,
		ExtSlice: []int{1, 2, 3},
	}
	data, _ := json.Marshal(ext)
	SetExtend("ext", data)
	HandleMOSNConfig(CfgTypeExtend, func(v interface{}) {
		es := v.([]v2.ExtendConfig)
		var raw json.RawMessage
		for _, ext := range es {
			if ext.Type == "ext" {
				raw = ext.Config
			}
		}
		if len(raw) == 0 {
			t.Fatal("no extend config stored")
		}
		got := &extConfig{}
		if err := json.Unmarshal(raw, got); err != nil {
			t.Fatalf("go data unexpected: %s", string(raw))
		}
		if !reflect.DeepEqual(got, ext) {
			t.Fatalf("got: %+v, expected: %+v", got, ext)
		}
	})
}

// TesttSetConfigConcurrency runs all SetXXConfig and HandleMOSNConfig
func TestSetConfigConcurrency(t *testing.T) {
	Reset()
	rand.Seed(time.Now().UnixNano())
	wg := sync.WaitGroup{}
	SetMosnConfig(&v2.MOSNConfig{
		Servers: []v2.ServerConfig{
			{
				ServerName: "server_name",
			},
		},
	})
	for _, fc := range []func(){
		func() {
			SetListenerConfig(v2.Listener{
				ListenerConfig: v2.ListenerConfig{
					Name: "test",
				},
			})
		},
		func() {
			SetClusterConfig(v2.Cluster{
				Name: "cluster_name",
			})
		},
		func() {
			SetRemoveClusterConfig("cluster_name")
		},
		func() {
			SetHosts("cluster_name", nil)
		},
		func() {
			SetRouter(v2.RouterConfiguration{
				RouterConfigurationConfig: v2.RouterConfigurationConfig{
					RouterConfigName: "test_router",
					RouterConfigPath: "/tmp/bench/test_router",
				},
			})
		},
		func() {
			SetExtend("test", json.RawMessage("123456"))
		},
		func() {
			SetClusterManagerTLS(v2.TLSConfig{
				Status: false,
			})
		},
		func() {
			DumpJSON()
		},
		func() {
			HandleMOSNConfig(CfgTypeMOSN, func(_ interface{}) {
				// do nothing
			})
		},
		func() {
			// dump data
			transferConfig()
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
