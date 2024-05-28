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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClusterMarshal(t *testing.T) {
	c := &Cluster{
		Name:                 "test",
		ClusterType:          SIMPLE_CLUSTER,
		LbType:               LB_RANDOM,
		MaxRequestPerConn:    10000,
		ConnBufferLimitBytes: 16 * 1024,
		LBSubSetConfig: LBSubsetConfig{
			FallBackPolicy: 1,
		},
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	nc := &Cluster{}
	if err := json.Unmarshal(b, nc); err != nil {
		t.Fatal(err)
	}
	if !(nc.Name == c.Name &&
		nc.ClusterType == c.ClusterType &&
		nc.LbType == c.LbType &&
		nc.MaxRequestPerConn == c.MaxRequestPerConn &&
		nc.ConnBufferLimitBytes == c.ConnBufferLimitBytes &&
		nc.LBSubSetConfig.FallBackPolicy == c.LBSubSetConfig.FallBackPolicy) {
		t.Error("unmarshal and marshal is not equal")
	}

}

func TestClusterUnmarshal(t *testing.T) {
	clusterConfig := `{
		"name": "test",
		"type": "SIMPLE",
		"lb_type": "LB_RANDOM",
		"circuit_breakers":[
			{
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
		],
		"lbconfig": {
			"choice_count": 10
  		}
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
		if !(breakers[0].MaxConnections == 10 &&
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
		"address": "127.0.0.1:8080",
		"type": "ingress",
		"connection_idle_timeout": "90s",
		"bind_port": true,
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
	ln := &Listener{}
	if err := json.Unmarshal(b, ln); err != nil {
		t.Error(err)
		return
	}
	if !(ln.Name == "test" &&
		ln.AddrConfig == "127.0.0.1:8080" &&
		ln.BindToPort == true &&
		ln.Type == INGRESS &&
		ln.ConnectionIdleTimeout.Duration == 90*time.Second &&
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
			fc.TLSContexts[0].Status == true) {
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
	// verify addr
	if ln.Addr == nil || ln.Addr.String() != "127.0.0.1:8080" {
		t.Error("listener addr is not expected")
	}
	if ln.PerConnBufferLimitBytes != 1<<15 {
		t.Error("listener buffer limit is not default value")
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

func TestProxyExtendConfigJson(t *testing.T) {
	configBytes := []byte(`
{
	"name": "testProxy",
	"extend_config": {
		"someInt": 1,
		"someFloat": 2.01,
		"someBool": true,
		"someString": "value"
	}
}
`)
	var config Proxy
	err := json.Unmarshal(configBytes, &config)
	assert.Nil(t, err)
	assert.Equal(t, int(config.ExtendConfig["someInt"].(float64)), 1)
	assert.Equal(t, float32(config.ExtendConfig["someFloat"].(float64)), float32(2.01))
	assert.Equal(t, config.ExtendConfig["someBool"].(bool), true)
	assert.Equal(t, config.ExtendConfig["someString"].(string), "value")
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
func TestDelayInjectUnmarshal(t *testing.T) {
	inject := `{
		"fixed_delay": "15s",
		"percentage": 100
	}`
	b := []byte(inject)
	di := &DelayInject{}
	if err := json.Unmarshal(b, di); err != nil {
		t.Error(err)
		return
	}
	if !(di.Delay == 15*time.Second && di.Percent == 100) {
		t.Error("delay inject failed")
	}
}
func TestStreamFaultInject(t *testing.T) {
	streamfilter := `{
		"delay": {
			"fixed_delay":"1s",
			"percentage": 100
		},
		"abort": {
			"status": 500,
			"percentage": 100
		},
		"upstream_cluster": "clustername",
		"headers": [
			{"name":"service","value":"test","regex":false},
			{"name":"user","value":"bob", "regex":false}
		]
	}`
	b := []byte(streamfilter)
	sfi := &StreamFaultInject{}
	if err := json.Unmarshal(b, sfi); err != nil {
		t.Error(err)
		return
	}
	if !(sfi.Delay.Delay == time.Second &&
		sfi.Delay.Percent == 100 &&
		sfi.Abort.Status == 500 &&
		sfi.Abort.Percent == 100 &&
		sfi.UpstreamCluster == "clustername" &&
		len(sfi.Headers) == 2) {
		t.Error("unexpected stream fault inject")
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
	p := &StreamProxy{}
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

func TestUDPProxyUnmarshal(t *testing.T) {
	udpproxy := `{
		"stat_prefix":"udp_proxy",
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

	b := []byte(udpproxy)
	p := &StreamProxy{}
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
