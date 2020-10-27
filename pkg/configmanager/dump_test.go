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
	"io/ioutil"
	"net"
	"os"
	"testing"

	v2 "mosn.io/mosn/pkg/config/v2"
)

// Test Dump Action with extensions
func TestDumpWithTransExtension(t *testing.T) {
	// Init
	Reset()
	// mock config path
	configPath = "/tmp/dump_test/with_transfer/mosn.json"
	os.RemoveAll("/tmp/dump_test/with_transfer")
	os.MkdirAll("/tmp/dump_test/with_transfer", 0755)
	// register a transfer extension
	RegisterTransferExtension(func(config *v2.MOSNConfig) {
		// clean some cluster hosts
		for idx, c := range config.ClusterManager.Clusters {
			if len(c.Spec.Subscribes) > 0 && c.Spec.Subscribes[0].Subscriber == "dynamic" {
				c.Hosts = nil
			}
			config.ClusterManager.Clusters[idx] = c
		}
	})
	defer RegisterTransferExtension(nil) // clean
	// mock update config
	// listener
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
	udpListener := v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       "udp_listener",
			BindToPort: true,
			Network:    "udp",
		},
		Addr: addr,
	}
	SetListenerConfig(udpListener)
	// clusters
	clusters := []v2.Cluster{
		// cluster with hosts
		{
			Name: "cluster_with_hosts",
		},
		// cluster hosts will be cleand
		{
			Name: "cluster_without_hosts",
			Spec: v2.ClusterSpecInfo{
				Subscribes: []v2.SubscribeSpec{
					{
						Subscriber: "dynamic",
					},
				},
			},
		},
	}
	for _, c := range clusters {
		SetClusterConfig(c)
	}
	// hossts
	hosts := map[string][]v2.Host{
		"cluster_with_hosts": []v2.Host{
			{
				HostConfig: v2.HostConfig{
					Address: "127.0.0.1:12345",
				},
			},
		},
		"cluster_without_hosts": []v2.Host{
			{
				HostConfig: v2.HostConfig{
					Address: "192.168.1.1:8080",
				},
			},
			{
				HostConfig: v2.HostConfig{
					Address: "192.168.1.2::8080",
				},
			},
		},
	}
	for cn, hs := range hosts {
		SetHosts(cn, hs)
	}
	// routers
	router := v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: "router00",
		},
	}
	SetRouter(router)
	// clustermanager.tls
	SetClusterManagerTLS(v2.TLSConfig{
		Status: true,
	})
	// Trigger a dump, dump all updated configs
	setDump()
	DumpConfig()
	// Verify
	mosnConfig := Load(configPath)
	server := mosnConfig.Servers[0]
	if !(len(server.Listeners) == 1 &&
		len(server.Routers) == 1 &&
		mosnConfig.ClusterManager.TLSContext.Status &&
		len(mosnConfig.ClusterManager.Clusters) == 2) {
		t.Fatalf("basic config dumped invalid: %+v", mosnConfig)
	}
	if !(server.Listeners[0].AddrConfig == "127.0.0.1:8080" && server.Listeners[0].Network == "udp") {
		t.Fatalf("server listener is invalid: %+v", server.Listeners[0])
	}
	read_clusters := mosnConfig.ClusterManager.Clusters
	for _, c := range read_clusters {
		switch c.Name {
		case "cluster_without_hosts":
			if len(c.Hosts) > 0 {
				t.Fatalf("cluster should have no hosts dumped")
			}
		case "cluster_with_hosts":
			if len(c.Hosts) == 0 {
				t.Fatalf("cluster should keeps hosts dumped")
			}
		default:
			t.Fatalf("invalid cluster name: %s", c.Name)
		}
	}

}

func TestYamlConfigDump(t *testing.T) {
	Reset()
	// mock config path
	configPath = "/tmp/dump_test/mosn.yaml"
	os.RemoveAll("/tmp/dump_test")
	os.MkdirAll("/tmp/dump_test", 0755)
	// test json marshal data
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	listener := v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       "listener",
			BindToPort: true,
		},
		Addr: addr, // should marshal to addr string
	}
	SetListenerConfig(listener)
	setDump()
	DumpConfig()
	mosnConfig := Load(configPath)
	if mosnConfig.Servers[0].Listeners[0].AddrConfig != "127.0.0.1:8080" {
		content, _ := ioutil.ReadFile(configPath)
		t.Fatalf("json to yaml is not expected: %+v", content)
	}
}
