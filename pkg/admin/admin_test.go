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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/juju/errors"
)

func getEffectiveConfig() (string, error) {
	resp, err := http.Get("http://localhost:8888/api/v1/config_dump")
	defer resp.Body.Close()

	if err != nil {
		return "", err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("call admin api failed response status: %d, %s", resp.StatusCode, string(b)))
	}

	if err != nil {
		return "", err
	}
	return string(b), nil
}

func TestSetConfig(t *testing.T) {
	type args struct {
		listenerConfigMap map[string]*v2.Listener
		clusterConfigMap  map[string]*v2.Cluster
	}

	listenerConf1 := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:                                  "test_listener",
			AddrConfig:                            "127.0.0.1:2045",
			BindToPort:                            true,
			HandOffRestoredDestinationConnections: false,
			LogPath:                               "stdout",
			FilterChains: []v2.FilterChain{
				{
					Filters: []v2.Filter{
						{
							Name: "proxy",
							Config: map[string]interface{}{
								"downstream_protocol": "Http1",
								"upstream_protocol":   "Http2",
								"virtual_hosts": []map[string]interface{}{
									{
										"name":    "clientHost",
										"domains": []interface{}{"*"},
										"routers": []map[string]interface{}{
											{
												"match": map[string]interface{}{
													"prefix": "/",
												},
												"route": map[string]interface{}{
													"cluster_name": "clientCluster",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	listenerConf2 := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:                                  "test_listener",
			AddrConfig:                            "127.0.0.1:2045",
			BindToPort:                            false,
			HandOffRestoredDestinationConnections: true,
			LogPath:                               "stdout",
			FilterChains: []v2.FilterChain{
				{
					Filters: []v2.Filter{
						{
							Name: "proxy",
							Config: map[string]interface{}{
								"downstream_protocol": "Http1",
								"upstream_protocol":   "Http2",
								"virtual_hosts": []map[string]interface{}{
									{
										"name":    "clientHost",
										"domains": []interface{}{"*"},
										"routers": []map[string]interface{}{
											{
												"match": map[string]interface{}{
													"prefix": "/xxx",
												},
												"route": map[string]interface{}{
													"cluster_name": "clientCluster",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	clusterConf1 := &v2.Cluster{
		Name:                 "clientCluster",
		ClusterType:          "SIMPLE",
		SubType:              "",
		LbType:               "LB_RANDOM",
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 32768,
		OutlierDetection:     v2.OutlierDetection{},
		HealthCheck: v2.HealthCheck{
			HealthCheckConfig: v2.HealthCheckConfig{
				Protocol: "",
				TimeoutConfig: v2.DurationConfig{
					Duration: 0,
				},
				IntervalConfig: v2.DurationConfig{
					Duration: 0,
				},
				IntervalJitterConfig: v2.DurationConfig{
					Duration: 0,
				},
			},
		},
		Spec: v2.ClusterSpecInfo{},
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy:  0,
			DefaultSubset:   nil,
			SubsetSelectors: nil,
		},
		TLS: v2.TLSConfig{},
		Hosts: []v2.Host{
			{
				HostConfig: v2.HostConfig{
					Address: "127.0.0.1:2046",
					Weight:  1,
					MetaDataConfig: v2.MetadataConfig{
						MetaKey: v2.LbMeta{
							LbMetaKey: nil,
						},
					},
				},
			},
		},
	}
	clusterConf2 := &v2.Cluster{
		Name:                 "clientCluster",
		ClusterType:          "SIMPLE",
		SubType:              "",
		LbType:               "LB_RANDOM",
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 32768,
		OutlierDetection:     v2.OutlierDetection{},
		HealthCheck: v2.HealthCheck{
			HealthCheckConfig: v2.HealthCheckConfig{
				Protocol: "",
				TimeoutConfig: v2.DurationConfig{
					Duration: 0,
				},
				IntervalConfig: v2.DurationConfig{
					Duration: 0,
				},
				IntervalJitterConfig: v2.DurationConfig{
					Duration: 0,
				},
			},
		},
		Spec: v2.ClusterSpecInfo{},
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy:  0,
			DefaultSubset:   nil,
			SubsetSelectors: nil,
		},
		TLS: v2.TLSConfig{},
		Hosts: []v2.Host{
			{
				HostConfig: v2.HostConfig{
					Address: "127.0.0.1:2045",
					Weight:  1,
					MetaDataConfig: v2.MetadataConfig{
						MetaKey: v2.LbMeta{
							LbMetaKey: nil,
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Add listener config",
			args: args{
				listenerConfigMap: map[string]*v2.Listener{
					"test_listener_1": listenerConf1,
				},
			},
			want: `{"cluster":{},"listener":{"test_listener":{"name":"test_listener","address":"127.0.0.1:2045","bind_port":true,"handoff_restoreddestination":false,"log_path":"stdout","filter_chains":[{"tls_context":{"status":false,"type":"","extend_verify":null},"filters":[{"type":"proxy","config":{"downstream_protocol":"Http1","upstream_protocol":"Http2","virtual_hosts":[{"domains":["*"],"name":"clientHost","routers":[{"match":{"prefix":"/"},"route":{"cluster_name":"clientCluster"}}]}]}}]}]}},"original_config":null}`,
		},
		{
			name: "Update listener config",
			args: args{
				listenerConfigMap: map[string]*v2.Listener{
					"test_listener_1": listenerConf1,
					"test_listener_2": listenerConf2,
				},
			},
			want: `{"cluster":{},"listener":{"test_listener":{"name":"test_listener","address":"127.0.0.1:2045","bind_port":true,"handoff_restoreddestination":false,"log_path":"stdout","filter_chains":[{"tls_context":{"status":false,"type":"","extend_verify":null},"filters":[{"type":"proxy","config":{"downstream_protocol":"Http1","upstream_protocol":"Http2","virtual_hosts":[{"domains":["*"],"name":"clientHost","routers":[{"match":{"prefix":"/xxx"},"route":{"cluster_name":"clientCluster"}}]}]}}]}]}},"original_config":null}`,
		},
		{
			name: "Add cluster config",
			args: args{
				clusterConfigMap: map[string]*v2.Cluster{
					"clientCluster_1": clusterConf1,
				},
			},
			want: `{"cluster":{"clientCluster":{"name":"clientCluster","type":"SIMPLE","sub_type":"","lb_type":"LB_RANDOM","max_request_per_conn":1024,"conn_buffer_limit_bytes":32768,"circuit_breakers":null,"outlier_detection":{"Consecutive5xx":0,"Interval":0,"BaseEjectionTime":0,"MaxEjectionPercent":0,"ConsecutiveGatewayFailure":0,"EnforcingConsecutive5xx":0,"EnforcingConsecutiveGatewayFailure":0,"EnforcingSuccessRate":0,"SuccessRateMinimumHosts":0,"SuccessRateRequestVolume":0,"SuccessRateStdevFactor":0},"health_check":{"protocol":"","timeout":"0s","interval":"0s","interval_jitter":"0s","healthy_threshold":0,"unhealthy_threshold":0},"spec":{},"lb_subset_config":{"fall_back_policy":0,"default_subset":null,"subset_selectors":null},"tls_context":{"status":false,"type":"","extend_verify":null},"hosts":[{"address":"127.0.0.1:2046","weight":1,"metadata":{"filter_metadata":{"mosn.lb":null}}}]}},"listener":{},"original_config":null}`,
		},
		{
			name: "Update cluster config",
			args: args{
				clusterConfigMap: map[string]*v2.Cluster{
					"clientCluster_1": clusterConf1,
					"clientCluster_2": clusterConf2,
				},
			},
			want: `{"cluster":{"clientCluster":{"name":"clientCluster","type":"SIMPLE","sub_type":"","lb_type":"LB_RANDOM","max_request_per_conn":1024,"conn_buffer_limit_bytes":32768,"circuit_breakers":null,"outlier_detection":{"Consecutive5xx":0,"Interval":0,"BaseEjectionTime":0,"MaxEjectionPercent":0,"ConsecutiveGatewayFailure":0,"EnforcingConsecutive5xx":0,"EnforcingConsecutiveGatewayFailure":0,"EnforcingSuccessRate":0,"SuccessRateMinimumHosts":0,"SuccessRateRequestVolume":0,"SuccessRateStdevFactor":0},"health_check":{"protocol":"","timeout":"0s","interval":"0s","interval_jitter":"0s","healthy_threshold":0,"unhealthy_threshold":0},"spec":{},"lb_subset_config":{"fall_back_policy":0,"default_subset":null,"subset_selectors":null},"tls_context":{"status":false,"type":"","extend_verify":null},"hosts":[{"address":"127.0.0.1:2045","weight":1,"metadata":{"filter_metadata":{"mosn.lb":null}}}]}},"listener":{},"original_config":null}`,
		},
	}

	srv := Start(nil)
	defer func() {
		if err := srv.Close(); err != nil {
			fmt.Errorf("server close error: %s", err)
		}
	}()
	time.Sleep(time.Second)

	var mux sync.Mutex

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux.Lock()
			Reset()
			defer func() {
				mux.Unlock()
			}()

			for _, v := range tt.args.listenerConfigMap {
				SetListenerConfig(v.Name, v)
			}
			for _, v := range tt.args.clusterConfigMap {
				SetClusterConfig(v.Name, v)
			}
			if got, err := getEffectiveConfig(); err != nil || strings.Compare(got, tt.want) != 0 {
				if err != nil {
					t.Errorf("getEffectiveConfig failed with error: %s", err)
				} else {
					t.Errorf("getEffectiveConfig() = %v, want %v", got, tt.want)
				}
			}
			Reset()
		})
	}
}
