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

package v2

import (
	"reflect"
	"testing"
	"time"
)

func TestClusterUnmarshal(t *testing.T) {
	clusterConfig := `{
		"name": "test",
		"type": "SIMPLE",
		"lb_type": "LB_RANDOM",
		"circuit_breakers":[
			{
				"priority":"HIGH",
				"max_connections":10,
				"max_retries":1
			}
		],
		"health_check": {
			"protocol":"http1",
			"timeout":"10s",
			"interval":"1m",
			"interval_jitter":"1m",
			"healthy_threshold":1,
			"service_name":"test"
		},
		"spec":{
			"subscribe":[
				{"service_name":"test"}
			]
		},
		"lb_subset_config":{
			"fall_back_policy":1,
			"default_subset":{
				"stage": "pre-release",
				"label": "gray"
			},
			"subset_selectors": [
				["stage", "type"]
			]
		},
		"tls_context":{
			"status":true
		},
		"hosts":[
			{
				"address": "127.0.0.1",
				"metadata": {
					"filter_metadata": {
						"mosn.lb": {
							"label": "gray"
						}
					}
				}
			}
		]
	}`
	b := []byte(clusterConfig)
	cluster := &Cluster{}
	if err := json.Unmarshal(b, cluster); err != nil {
		t.Error(err)
		return
	}

	if !(cluster.Name == "test" &&
		cluster.ClusterType == SIMPLE_CLUSTER &&
		cluster.LbType == LB_RANDOM) {
		t.Error("basic verify failed")
	}
	breakers := cluster.CirBreThresholds.Thresholds
	if len(breakers) != 1 {
		t.Error("CirBreThresholds failed")
	} else {
		if !(breakers[0].Priority == HIGH &&
			breakers[0].MaxConnections == 10 &&
			breakers[0].MaxRetries == 1) {
			t.Error("CirBreThresholds failed")
		}
	}
	healthcheck := cluster.HealthCheck
	if !(healthcheck.Protocol == "http1" &&
		healthcheck.Timeout == 10*time.Second &&
		healthcheck.Interval == time.Minute &&
		healthcheck.IntervalJitter == time.Minute) {
		t.Error("healthcheck failed")
	}
	spec := cluster.Spec
	if len(spec.Subscribes) != 1 {
		t.Error("spec failed")
	} else {
		if spec.Subscribes[0].ServiceName != "test" {
			t.Error("spec failed")
		}
	}
	lbsub := cluster.LBSubSetConfig
	if !(lbsub.FallBackPolicy == 1 &&
		len(lbsub.DefaultSubset) == 2 &&
		len(lbsub.SubsetSelectors) == 1 &&
		len(lbsub.SubsetSelectors[0]) == 2) {
		t.Error("lbsubset failed")
	}
	if !cluster.TLS.Status {
		t.Error("tls failed")
	}
	hosts := cluster.Hosts
	if len(hosts) != 1 {
		t.Error("hosts failed")
	} else {
		host := hosts[0]
		if host.Address != "127.0.0.1" {
			t.Error("hosts failed")
		}
		meta := host.MetaData
		if v, ok := meta["label"]; !ok {
			t.Error("hosts failed")
		} else {
			if v != "gray" {
				t.Error("hosts failed")
			}
		}
	}
}

func TestListenerUnmarshal(t *testing.T) {
	lc := `{
		"name": "test",
		"address": "127.0.0.1",
		"bind_port": true,
		"handoff_restoreddestination": true,
		"log_path": "stdout",
		"log_level": "TRACE",
		"access_logs": [
			{
				"log_path":"stdout"
			}
		],
		"filter_chains": [
			{
				"match": "test",
				"tls_context":{
					"status":true
				},
				"filters":[
					{
						"type":"proxy",
						"config": {
							"test":"test"
						}
					}
				]
			}
		],
		"stream_filters": [
			{
				"type": "test",
				"config": {
					"test":"test"
				}
			}
		],
		"inspector": true
	}`
	b := []byte(lc)
	ln := &ListenerConfig{}
	if err := json.Unmarshal(b, ln); err != nil {
		t.Error(err)
		return
	}
	if !(ln.Name == "test" &&
		ln.AddrConfig == "127.0.0.1" &&
		ln.BindToPort == true &&
		ln.HandOffRestoredDestinationConnections == true &&
		ln.LogPath == "stdout" &&
		ln.LogLevelConfig == "TRACE" &&
		ln.Inspector == true) {
		t.Error("listener basic failed")
	}
	if len(ln.AccessLogs) != 1 || ln.AccessLogs[0].Path != "stdout" {
		t.Error("listener accesslog failed")
	}
	if len(ln.FilterChains) != 1 {
		t.Error("listener filterchains failed")
	} else {
		fc := ln.FilterChains[0]
		if !(fc.FilterChainMatch == "test" &&
			fc.TLS.Status == true) {
			t.Error("listener filterchains failed")
		}
		if len(fc.Filters) != 1 || fc.Filters[0].Type != "proxy" {
			t.Error("listener filterchains failed")
		}
	}
	if len(ln.StreamFilters) != 1 {
		sf := ln.StreamFilters[0]
		if sf.Type != "test" {
			t.Error("listener stream filter failed")
		}
	}
}

func TestRouterConfigUmaeshal(t *testing.T) {
	routerConfig := `{
		"router_config_name":"test_router",
		"virtual_hosts": [
			{
				"name": "vitrual",
				"domains":["*"],
				"virtual_clusters":[
					{
						"name":"vc",
						"pattern":"test"
					}
				],
				"routers":[
					{
						"match": {
							"prefix":"/",
							"runtime": {
								"default_value":10,
								"runtime_key":"test"
							},
							"headers":[
								{
									"name":"service",
									"value":"test"
								}
							]
						},
						"route":{
							"cluster_name":"cluster",
							"weighted_clusters": [
								{
									"cluster": {
										"name": "test",
										"weight":100,
										"metadata_match": {
											"filter_metadata": {
												"mosn.lb": {
													"test":"test"
												}
											}
										}
									}
								}
							],
							"metadata_match": {
								"filter_metadata": {
									"mosn.lb": {
										"test":"test"
									}
								}
							},
							"timeout": "1s",
							"retry_policy":{
								"retry_on": true,
								"retry_timeout": "1m",
								"num_retries":10
							}
						},
						"redirect":{
							"host_redirect": "test",
							"response_code": 302
						},
						"metadata":{
							"filter_metadata": {
								"mosn.lb": {
									 "test":"test"
								}
							}
						},
						"decorator":"test"
					}
				]
			}
		]
	}`

	bytes := []byte(routerConfig)
	router := &RouterConfiguration{}

	if err := json.Unmarshal(bytes, router); err != nil {
		t.Error(err)
		return
	}

	if len(router.VirtualHosts) != 1 {
		t.Error("virtual host failed")
	} else {
		vh := router.VirtualHosts[0]
		if !(vh.Name != "virtual" &&
			len(vh.Domains) == 1 &&
			vh.Domains[0] == "*" &&
			len(vh.VirtualClusters) == 1 &&
			vh.VirtualClusters[0].Name == "vc" &&
			vh.VirtualClusters[0].Pattern == "test") {
			t.Error("virtual host failed")
		}
		if len(vh.Routers) != 1 {
			t.Error("virtual host failed")
		} else {
			router := vh.Routers[0]
			if !(router.Match.Prefix == "/" &&
				router.Match.Runtime.DefaultValue == 10 &&
				router.Match.Runtime.RuntimeKey == "test" &&
				len(router.Match.Headers) == 1 &&
				router.Match.Headers[0].Name == "service" &&
				router.Match.Headers[0].Value == "test") {
				t.Error("virtual host failed")
			}
			meta := Metadata{
				"test": "test",
			}
			if !(router.Route.ClusterName == "cluster" &&
				router.Route.Timeout == time.Second &&
				router.Route.RetryPolicy.RetryTimeout == time.Minute &&
				router.Route.RetryPolicy.RetryOn == true &&
				router.Route.RetryPolicy.NumRetries == 10 &&
				reflect.DeepEqual(meta, router.Metadata) &&
				router.Decorator == "test") {
				t.Error("virtual host failed")
			}
			if len(router.Route.WeightedClusters) != 1 {
				t.Error("virtual host failed")
			} else {
				wc := router.Route.WeightedClusters[0]
				if !(wc.Cluster.Name == "test" &&
					wc.Cluster.Weight == 100 &&
					reflect.DeepEqual(meta, wc.Cluster.MetadataMatch)) {
					t.Error("virtual host failed")
				}
			}
			if !(router.Redirect.HostRedirect == "test" &&
				router.Redirect.ResponseCode == 302) {
				t.Error("virtual host failed")
			}

		}
	}

}

func TestProxyUnmarshal(t *testing.T) {
	proxy := `{
		"name": "proxy",
		"downstream_protocol": "Http1",
		"upstream_protocol": "Sofarpc",
		"extend_config":{
			"sub_protocol":"example"
		}
	}`
	b := []byte(proxy)
	p := &Proxy{}
	if err := json.Unmarshal(b, p); err != nil {
		t.Error(err)
		return
	}
	if !(p.Name == "proxy" &&
		p.DownstreamProtocol == "Http1" &&
		p.UpstreamProtocol == "Sofarpc") {
		t.Error("baisc failed")
	}
}
func TestFaultInjectUnmarshal(t *testing.T) {
	fault := `{
		"delay_percent": 100,
		"delay_duration": "15s"
	}`
	b := []byte(fault)
	fi := &FaultInject{}
	if err := json.Unmarshal(b, fi); err != nil {
		t.Error(err)
		return
	}
	if !(fi.DelayDuration == uint64(15*time.Second) && fi.DelayPercent == 100) {
		t.Error("fault inject failed")
	}
}

func TestTCPProxyUnmarshal(t *testing.T) {
	tcpproxy := `{
		"stat_prefix":"tcp_proxy",
		"cluster":"cluster",
		"max_connect_attempts":1000,
		"routes":[
			{
				"cluster": "test",
				"SourceAddrs": [
					{
						"address":"127.0.0.1",
						"length":32
					}
				],
				"DestinationAddrs":[
					{
						"address":"127.0.0.1",
						"length":32
					}
				],
				"SourcePort":"8080",
				"DestinationPort":"8080"
			}
		]
	}`
	b := []byte(tcpproxy)
	p := &TCPProxy{}
	if err := json.Unmarshal(b, p); err != nil {
		t.Error(err)
		return
	}
	if len(p.Routes) != 1 {
		t.Error("route failed")
	} else {
		r := p.Routes[0]
		if !(r.Cluster == "test" &&
			len(r.SourceAddrs) == 1 &&
			r.SourceAddrs[0].Address == "127.0.0.1" &&
			r.SourceAddrs[0].Length == 32 &&
			len(r.DestinationAddrs) == 1 &&
			r.DestinationAddrs[0].Address == "127.0.0.1" &&
			r.DestinationAddrs[0].Length == 32 &&
			r.SourcePort == "8080" &&
			r.DestinationPort == "8080") {
			t.Error("route failed")
		}
	}
}
func TestHealthCheckFilterUnmarshal(t *testing.T) {
	hc := `{
		"passthrough":true,
		"cache_time":"10m",
		"endpoint": "test",
		"cluster_min_healthy_percentages":{
			"test":10.0
		}
	}`
	b := []byte(hc)
	filter := &HealthCheckFilter{}
	if err := json.Unmarshal(b, filter); err != nil {
		t.Error(err)
		return
	}
	if !(filter.PassThrough &&
		filter.CacheTime == 10*time.Minute &&
		filter.Endpoint == "test" &&
		len(filter.ClusterMinHealthyPercentage) == 1 &&
		filter.ClusterMinHealthyPercentage["test"] == 10.0) {
		t.Error("health check filter failed")
	}
}

func TestServiceRegistryInfoUnmarshal(t *testing.T) {
	sri := `{
		"application": {
			"ant_share_cloud":true
		},
		"publish_info":[
			{
				"service_name": "test",
				"pub_data": "foo"
			}
		]
	}`
	b := []byte(sri)
	info := &ServiceRegistryInfo{}
	if err := json.Unmarshal(b, info); err != nil {
		t.Error(err)
		return
	}
	if !(info.ServiceAppInfo.AntShareCloud &&
		len(info.ServicePubInfo) == 1 &&
		info.ServicePubInfo[0].Pub.ServiceName == "test" &&
		info.ServicePubInfo[0].Pub.PubData == "foo") {
		t.Error("service registry info failed")
	}
}
