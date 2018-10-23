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

package admin

import (
	"errors"
	"fmt"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

type checkFunction func() error

func setupSubTest(t *testing.T) func(t *testing.T) {
	return func(t *testing.T) {
		Reset()
	}
}

func TestSetListenerConfig(t *testing.T) {
	cases := []struct {
		name      string
		listeners []v2.Listener
		expect    checkFunction
	}{
		{
			name: "add",
			listeners: []v2.Listener{
				{
					ListenerConfig: v2.ListenerConfig{
						Name:       "test",
						BindToPort: false,
						LogPath:    "stdout",
					},
				},
			},
			expect: func() error {
				if len(conf.Listener) != 1 && conf.Listener["test"].Name != "test" {
					return errors.New("listener add failed")
				}
				if len(conf.Listener["test"].FilterChains) != 0 {
					return errors.New("listener add failed, FilterChains should be empty")
				}
				return nil
			},
		},
		{
			name: "update",
			listeners: []v2.Listener{
				{
					ListenerConfig: v2.ListenerConfig{
						Name:       "test",
						BindToPort: false,
						LogPath:    "stdout",
					},
				},
				{
					ListenerConfig: v2.ListenerConfig{
						Name:       "test",
						BindToPort: false,
						LogPath:    "stdout",
						FilterChains: []v2.FilterChain{
							{
								FilterChainMatch: "",
								Filters: []v2.Filter{
									{
										Type:   "xxx",
										Config: nil,
									},
								},
							},
						},
					},
				},
			},
			expect: func() error {
				if len(conf.Listener) != 1 && conf.Listener["test"].Name != "test" {
					return errors.New("listener add failed")
				}
				if len(conf.Listener["test"].FilterChains) != 1 {
					return errors.New("listener add failed, FilterChains count should be one")
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
				SetListenerConfig(listener.ListenerConfig.Name, listener)
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
							MetaDataConfig: v2.MetadataConfig{
								MetaKey: v2.LbMeta{
									LbMetaKey: nil,
								},
							},
						},
					},
				},
			},
			expect: func() error {
				if len(conf.Cluster) != 1 || len(conf.Cluster["outbound|9080||productpage.default.svc.cluster.local"].Hosts) != 1 {
					return errors.New("[outbound|9080||productpage.default.svc.cluster.local] not exists or empty")
				}
				hosts := conf.Cluster["outbound|9080||productpage.default.svc.cluster.local"].Hosts
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
				if len(conf.Cluster) != 1 {
					return errors.New("[outbound|9080||productpage.default.svc.cluster.local] not exists or empty")
				}
				lbType := conf.Cluster["outbound|9080||productpage.default.svc.cluster.local"].LbType
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
								MetaDataConfig: v2.MetadataConfig{
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
							MetaDataConfig: v2.MetadataConfig{
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
							MetaDataConfig: v2.MetadataConfig{
								MetaKey: v2.LbMeta{
									LbMetaKey: nil,
								},
							},
						},
					},
				},
			},
			expect: func() error {
				if len(conf.Cluster) != 1 || len(conf.Cluster["outbound|9080||productpage.default.svc.cluster.local"].Hosts) != 2 {
					return errors.New("[outbound|9080||productpage.default.svc.cluster.local] not exists or hosts number error")
				}
				hosts := conf.Cluster["outbound|9080||productpage.default.svc.cluster.local"].Hosts
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
				SetClusterConfig(cluster.Name, cluster)
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

func BenchmarkSetListenerConfig_Add(b *testing.B) {
	listener := v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       "test",
			BindToPort: false,
			LogPath:    "stdout",
			FilterChains: []v2.FilterChain{
				{
					FilterChainMatch: "",
					Filters: []v2.Filter{
						{
							Type:   "xxx",
							Config: nil,
						},
					},
				},
			},
		},
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		num := i % 100
		SetListenerConfig(string(num), listener)
	}
	Reset()
}

func BenchmarkDump(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Dump()
	}
	Reset()
}
