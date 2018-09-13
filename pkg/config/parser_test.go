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

package config

import (
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
)

type testCallback struct {
	Count int
}

func (t *testCallback) ParsedCallback(data interface{}, endParsing bool) error {
	t.Count++
	return nil
}

var cb testCallback

func TestMain(m *testing.M) {
	RegisterConfigParsedListener(ParseCallbackKeyCluster, cb.ParsedCallback)
	RegisterConfigParsedListener(ParseCallbackKeyServiceRgtInfo, cb.ParsedCallback)
	m.Run()
}

func TestParseClusterConfig(t *testing.T) {
	// Test Host Weight trans and hosts maps return
	cb.Count = 0
	cluster := v2.Cluster{
		Name: "test",
		Hosts: []v2.Host{
			{
				HostConfig: v2.HostConfig{
					Address: "127.0.0.1:80",
					Weight:  200,
				},
			},
			{
				HostConfig: v2.HostConfig{
					Address: "127.0.0.1:8080",
					Weight:  0,
				},
			},
			{
				HostConfig: v2.HostConfig{
					Address: "127.0.0.1:8080",
					Weight:  100,
				},
			},
		},
	}
	clusters, clMap := ParseClusterConfig([]v2.Cluster{cluster})
	c := clusters[0]
	for _, h := range c.Hosts {
		if h.Weight < MinHostWeight || h.Weight > MaxHostWeight {
			t.Error("unexpected host weight")
		}
	}
	if hosts, ok := clMap["test"]; !ok {
		t.Error("no cluster map")
	} else {
		if !reflect.DeepEqual(hosts, c.Hosts) {
			t.Error("unexpected hosts map")
		}
	}
	if cb.Count != 1 {
		t.Error("no callback")
	}

}
func TestParseListenerConfig(t *testing.T) {
	// test listener inherit replace exists
	// make inherit listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Error(err)
		return
	}
	defer listener.Close()
	tcpListener := listener.(*net.TCPListener)
	inherit := &v2.Listener{
		Addr:            tcpListener.Addr(),
		InheritListener: tcpListener,
	}

	lc := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			AddrConfig:     tcpListener.Addr().String(),
			LogLevelConfig: "debug",
		},
	}
	ln := ParseListenerConfig(lc, []*v2.Listener{inherit})
	if !(ln.Addr != nil &&
		ln.Addr.String() == tcpListener.Addr().String() &&
		ln.PerConnBufferLimitBytes == 1<<15 &&
		ln.InheritListener != nil &&
		ln.LogLevel == uint8(log.DEBUG)) {
		t.Error("listener parse unexpected")
		t.Log(ln.Addr.String(), ln.InheritListener != nil, ln.LogLevel)
	}
	if !inherit.Remain {
		t.Error("no inherit listener")
	}
}

func TestParseProxyFilter(t *testing.T) {
	proxyConfigStr := `{
		"name": "proxy",
		"downstream_protocol": "sofarpc",
		"upstream_protocol": "http2",
		"virtual_hosts": [
			{
				"name": "vitrual",
				"domains":["*"],
				"routers":[
					{
						"match": {"prefix":"/"},
					 	"route":{
							 "cluster_name":"cluster"
					 	}
					}
				]
			}
		],
		"extend_config":{
			"sub_protocol":"example"
		}
	}`
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(proxyConfigStr), &m); err != nil {
		t.Error(err)
		return
	}
	proxy := ParseProxyFilter(m)
	if !(proxy.Name == "proxy" &&
		proxy.DownstreamProtocol == string(protocol.SofaRPC) &&
		proxy.UpstreamProtocol == string(protocol.HTTP2)) {
		t.Error("parse proxy filter failed")
	}
}

func TestParseFaultInjectFilter(t *testing.T) {
	m := map[string]interface{}{
		"delay_percent":  100,
		"delay_duration": "15s",
	}
	faultInject := ParseFaultInjectFilter(m)
	if !(faultInject.DelayDuration == uint64(15*time.Second) && faultInject.DelayPercent == 100) {
		t.Error("parse fault inject failed")
	}
}

func TestParseHealthCheckFilter(t *testing.T) {
	m := map[string]interface{}{
		"passthrough": true,
		"cache_time":  "10m",
		"endpoint":    "test",
		"cluster_min_healthy_percentages": map[string]float64{
			"test": 10.0,
		},
	}
	healthCheck := ParseHealthCheckFilter(m)
	if !(healthCheck.PassThrough &&
		healthCheck.CacheTime == 10*time.Minute &&
		healthCheck.Endpoint == "test" &&
		len(healthCheck.ClusterMinHealthyPercentage) == 1 &&
		healthCheck.ClusterMinHealthyPercentage["test"] == 10.0) {
		t.Error("parse health check filter failed")
	}
}

func TestParseTCPProxy(t *testing.T) {
	m := map[string]interface{}{
		"routes": []interface{}{
			map[string]interface{}{
				"cluster":           "test",
				"source_addrs":      []string{"127.0.0.1:80"},
				"destination_addrs": []string{"127.0.0.1:8080"},
			},
		},
	}
	tcpproxy, err := ParseTCPProxy(m)
	if err != nil {
		t.Error(err)
		return
	}
	if len(tcpproxy.Routes) != 1 {
		t.Error("parse tcpproxy failed")
	} else {
		r := tcpproxy.Routes[0]
		if !(r.Cluster == "test" &&
			len(r.SourceAddrs) == 1 &&
			r.SourceAddrs[0].String() == "127.0.0.1:80" &&
			len(r.DestinationAddrs) == 1 &&
			r.DestinationAddrs[0].String() == "127.0.0.1:8080") {
			t.Error("parse tcpproxy failed")
		}
	}
}
func TestParseServiceRegistry(t *testing.T) {
	cb.Count = 0
	ParseServiceRegistry(v2.ServiceRegistryInfo{})
	if cb.Count != 1 {
		t.Error("no callback")
	}
}
