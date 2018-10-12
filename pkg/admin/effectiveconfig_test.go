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
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

func setupSubTest(t *testing.T) func(t *testing.T) {
	return func(t *testing.T) {
		Reset()
	}
}

func TestSetListenerConfig_And_Dump(t *testing.T) {
	cases := []struct {
		name      string
		listeners []v2.Listener
		expect    string
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
			expect: `{"listener":{"test":{"name":"test","address":"","bind_port":false,"handoff_restoreddestination":false,"log_path":"stdout","filter_chains":null}}}`,
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
			expect: `{"listener":{"test":{"name":"test","address":"","bind_port":false,"handoff_restoreddestination":false,"log_path":"stdout","filter_chains":[{"tls_context":{"status":false,"type":""},"filters":[{"type":"xxx"}]}]}}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tearDownSubTest := setupSubTest(t)
			defer tearDownSubTest(t)

			for _, listener := range tc.listeners {
				SetListenerConfig(listener.ListenerConfig.Name, listener)
			}
			if buf, err := Dump(); err != nil {
				t.Error(err)
			} else {
				actual := string(buf)
				if actual != tc.expect {
					t.Errorf("ListenerConfig set/dump failed\nexpect: %s\nactual: %s", tc.expect, actual)
				}
			}
		})
	}
}

func TestSetClusterAndHosts(t *testing.T) {
	cases := []struct {
		name     string
		clusters []v2.Cluster
		hosts    map[string][]v2.Host
		expect   string
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
			expect: `{"cluster":{"outbound|9080||productpage.default.svc.cluster.local":{"name":"outbound|9080||productpage.default.svc.cluster.local","type":"EDS","sub_type":"","lb_type":"LB_ROUNDROBIN","max_request_per_conn":0,"conn_buffer_limit_bytes":0,"circuit_breakers":null,"outlier_detection":{"Consecutive5xx":0,"Interval":0,"BaseEjectionTime":0,"MaxEjectionPercent":0,"ConsecutiveGatewayFailure":0,"EnforcingConsecutive5xx":0,"EnforcingConsecutiveGatewayFailure":0,"EnforcingSuccessRate":0,"SuccessRateMinimumHosts":0,"SuccessRateRequestVolume":0,"SuccessRateStdevFactor":0},"health_check":{"protocol":"","timeout":"0s","interval":"0s","interval_jitter":"0s","healthy_threshold":0,"unhealthy_threshold":0},"spec":{},"lb_subset_config":{"fall_back_policy":0,"default_subset":null,"subset_selectors":null},"tls_context":{"status":false,"type":""},"hosts":[{"address":"172.16.1.154:9080","weight":1,"metadata":{"filter_metadata":{"mosn.lb":null}}}]}}}`,
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
			hosts:  map[string][]v2.Host{},
			expect: `{"cluster":{"outbound|9080||productpage.default.svc.cluster.local":{"name":"outbound|9080||productpage.default.svc.cluster.local","type":"EDS","sub_type":"","lb_type":"LB_RANDOM","max_request_per_conn":0,"conn_buffer_limit_bytes":0,"circuit_breakers":null,"outlier_detection":{"Consecutive5xx":0,"Interval":0,"BaseEjectionTime":0,"MaxEjectionPercent":0,"ConsecutiveGatewayFailure":0,"EnforcingConsecutive5xx":0,"EnforcingConsecutiveGatewayFailure":0,"EnforcingSuccessRate":0,"SuccessRateMinimumHosts":0,"SuccessRateRequestVolume":0,"SuccessRateStdevFactor":0},"health_check":{"protocol":"","timeout":"0s","interval":"0s","interval_jitter":"0s","healthy_threshold":0,"unhealthy_threshold":0},"spec":{},"lb_subset_config":{"fall_back_policy":0,"default_subset":null,"subset_selectors":null},"tls_context":{"status":false,"type":""},"hosts":null}}}`,
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
			expect: `{"cluster":{"outbound|9080||productpage.default.svc.cluster.local":{"name":"outbound|9080||productpage.default.svc.cluster.local","type":"EDS","sub_type":"","lb_type":"LB_ROUNDROBIN","max_request_per_conn":0,"conn_buffer_limit_bytes":0,"circuit_breakers":null,"outlier_detection":{"Consecutive5xx":0,"Interval":0,"BaseEjectionTime":0,"MaxEjectionPercent":0,"ConsecutiveGatewayFailure":0,"EnforcingConsecutive5xx":0,"EnforcingConsecutiveGatewayFailure":0,"EnforcingSuccessRate":0,"SuccessRateMinimumHosts":0,"SuccessRateRequestVolume":0,"SuccessRateStdevFactor":0},"health_check":{"protocol":"","timeout":"0s","interval":"0s","interval_jitter":"0s","healthy_threshold":0,"unhealthy_threshold":0},"spec":{},"lb_subset_config":{"fall_back_policy":0,"default_subset":null,"subset_selectors":null},"tls_context":{"status":false,"type":""},"hosts":[{"address":"172.16.1.154:9080","weight":1,"metadata":{"filter_metadata":{"mosn.lb":null}}},{"address":"172.16.1.155:9080","weight":3,"metadata":{"filter_metadata":{"mosn.lb":null}}}]}}}`,
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

			if buf, err := Dump(); err != nil {
				t.Error(err)
			} else {
				actual := string(buf)
				if actual != tc.expect {
					t.Errorf("ListenerConfig set/dump failed\nexpect: %s\nactual: %s", tc.expect, actual)
				}
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
