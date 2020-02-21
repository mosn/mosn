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

package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/featuregate"
)

func TestFeatureDump(t *testing.T) {
	f := &ConfigAutoFeature{
		BaseFeatureSpec: featuregate.BaseFeatureSpec{
			DefaultValue: true,
		},
	}
	Reset()
	createMosnConfig()
	if cfg := configmanager.Load(testConfigPath); cfg != nil {
		SetMosnConfig(cfg)
		// init set
		ln := cfg.Servers[0].Listeners[0]
		SetListenerConfig(ln.Name, ln)
		cluster := cfg.ClusterManager.Clusters[0]
		SetClusterConfig(cluster.Name, cluster)
		router := cfg.Servers[0].Routers[0]
		SetRouter(router.RouterConfigName, *router)

	}
	// update config
	SetListenerConfig("test_mosn_listener", v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       "test_mosn_listener",
			AddrConfig: "127.0.0.1:8080",
		},
	})
	SetClusterConfig("cluster002", v2.Cluster{
		Name: "cluster002",
	})
	SetRouter("test_router_2", v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: "test_router_2",
			RouterConfigPath: "/tmp/routers/test_router_2/",
		},
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{
				Name: "virtualhost_1",
			},
		},
	})
	f.dumpConfig()
	f.doDumpConfig()
	// call config dump really
	configmanager.DumpConfig()
	// verify
	cfg := configmanager.Load(testConfigPath)
	b, err := Dump()
	if err != nil {
		t.Fatal(err)
	}
	output := string(b)

	// config verify
	if !(len(cfg.Servers[0].Listeners) == 1 &&
		len(cfg.Servers[0].Listeners[0].FilterChains) == 0) {
		t.Fatal("listener in dump file is not expected")
	}
	if len(cfg.ClusterManager.Clusters) != 2 {
		t.Fatalf("clusters in dump file is not expected")
	}
	clusters, err := ioutil.ReadDir("/tmp/clusters")
	if err != nil || len(clusters) != 2 {
		t.Fatalf("cluster is not dumped into directory")
	}
	if len(cfg.Servers[0].Routers) != 2 {
		t.Fatalf("routers in dump file is not expected")
	}
	if routers, err := ioutil.ReadDir("/tmp/routers/test_router"); err != nil || len(routers) != 1 {
		t.Fatalf("router is not dumped into directory")
	}
	if routers, err := ioutil.ReadDir("/tmp/routers/test_router_2"); err != nil || len(routers) != 1 {
		t.Fatalf("router is not dumped into directory")
	}

	// should have cluster and router in plain text
	if !(strings.Contains(output, "virtualhost") &&
		strings.Contains(output, "lb_type")) {
		t.Fatalf("output is not expected: %s", output)
	}
	// should have other configs
	if !(strings.Contains(output, "socket_address") && // admin
		strings.Contains(output, "tracing") && // tracing
		strings.Contains(output, "metrics") && // metrics
		strings.Contains(output, "pprof")) {
		t.Fatalf("output is not expected: %s", output)
	}

}

const testConfigPath = "/tmp/mosn_admin.json"

func createMosnConfig() {
	routerPath := "/tmp/routers/test_router/"
	clusterPath := "/tmp/clusters"

	os.Remove(testConfigPath)
	os.RemoveAll(clusterPath)
	os.MkdirAll(clusterPath, 0755)
	os.RemoveAll(routerPath)
	os.MkdirAll(routerPath, 0755)

	ioutil.WriteFile(testConfigPath, []byte(mosnConfig), 0644)
	// write router
	ioutil.WriteFile(fmt.Sprintf("%s/virtualhost00.json", routerPath), []byte(`{
		"name": "virtualhost_0"
	}`), 0644)
	// write cluster
	ioutil.WriteFile(fmt.Sprintf("%s/cluster001.json", clusterPath), []byte(`{
		"name": "cluster001",
		"type": "SIMPLE",
		"lb_type": "LB_RANDOM"
	}`), 0644)

}

const mosnConfig = `{
	"servers": [
                {
                        "mosn_server_name": "test_mosn_server",
                        "listeners": [
                                {
                                         "name": "test_mosn_listener",
                                         "address": "127.0.0.1:8080",
                                         "filter_chains": [
                                                {
                                                        "filters": [
                                                                {
                                                                         "type": "proxy",
                                                                         "config": {
                                                                                 "downstream_protocol": "SofaRpc",
                                                                                 "upstream_protocol": "SofaRpc",
                                                                                 "router_config_name": "test_router"
                                                                         }
                                                                }
                                                        ]
                                                }
                                         ],
                                         "stream_filters": [
                                         ]
                                }
                        ],
                        "routers": [
                                {
                                        "router_config_name": "test_router",
                                        "router_configs": "/tmp/routers/test_router/"
                                }
                        ]
                }
         ],
         "cluster_manager": {
                 "clusters_configs": "/tmp/clusters"
         },
         "admin": {
                 "address": {
                         "socket_address": {
                                 "address": "0.0.0.0",
                                 "port_value": 34901
                         }
                 }
         }

}`
