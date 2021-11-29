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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
)

// test marshal, use unmarshal to verify it
func TestHealthCheckMarshal(t *testing.T) {
	hc := &HealthCheck{
		HealthCheckConfig: HealthCheckConfig{
			Protocol:           "test",
			HealthyThreshold:   1,
			UnhealthyThreshold: 2,
			ServiceName:        "test",
			CommonCallbacks:    []string{"test"},
		},
		Timeout:        time.Second,
		Interval:       time.Second,
		IntervalJitter: time.Second,
	}
	b, err := json.Marshal(hc)
	if err != nil {
		t.Fatal(err)
	}
	nhc := &HealthCheck{}
	if err := json.Unmarshal(b, nhc); err != nil {
		t.Fatal(err)
	}
	if !(hc.Protocol == nhc.Protocol &&
		hc.HealthyThreshold == nhc.HealthyThreshold &&
		hc.UnhealthyThreshold == nhc.UnhealthyThreshold &&
		hc.ServiceName == nhc.ServiceName &&
		len(hc.CommonCallbacks) == len(nhc.CommonCallbacks) &&
		hc.CommonCallbacks[0] == nhc.CommonCallbacks[0] &&
		hc.Timeout == nhc.Timeout &&
		hc.Interval == nhc.Interval &&
		hc.IntervalJitter == nhc.IntervalJitter) {
		t.Errorf("unmarshal and marshal is not equal old: %v, new: %v", hc, nhc)
	}
}

// json_test.go test the json marshaler and unmarshaler implementation
func TestHealthCheckUnmarshal(t *testing.T) {
	cfgStr := `{
		"protocol": "test",
		"healthy_threshold": 1,
		"unhealthy_threshold": 2,
		"service_name": "test",
		"common_callbacks": ["test"],
		"timeout":"1s",
		"interval":"1s",
		"interval_jitter":"1s"
	}`
	hc := &HealthCheck{}
	if err := json.Unmarshal([]byte(cfgStr), hc); err != nil {
		t.Fatal(err)
	}
	if !(hc.Protocol == "test" &&
		hc.HealthyThreshold == 1 &&
		hc.UnhealthyThreshold == 2 &&
		hc.ServiceName == "test" &&
		len(hc.CommonCallbacks) == 1 &&
		hc.Timeout == time.Second &&
		hc.Interval == time.Second &&
		hc.IntervalJitter == time.Second) {
		t.Error("unmarshal unexpected")
	}
}

func TestHostMarshal(t *testing.T) {
	host := &Host{
		MetaData: map[string]string{
			"label": "gray",
		},
	}
	b, err := json.Marshal(host)
	if err != nil {
		t.Fatal(err)
	}
	nhost := &Host{}
	if err := json.Unmarshal(b, nhost); err != nil {
		t.Fatal(err)
	}
	if v, ok := nhost.MetaData["label"]; !ok || v != "gray" {
		t.Fatal("unmarshal result is not expected")
	}
}

func TestHostUnmarshal(t *testing.T) {
	cfgStr := `{
			"metadata": {
				"filter_metadata": {
					"mosn.lb": {
						"label": "gray"
					}
				}
			}
		}`
	host := &Host{}
	if err := json.Unmarshal([]byte(cfgStr), host); err != nil {
		t.Fatal(err)
	}
	if v, ok := host.MetaData["label"]; !ok || v != "gray" {
		t.Fatal("unmarshal result is not expected")
	}
}

func TestHealthCheckFilterMarshal(t *testing.T) {
	hc := &HealthCheckFilter{
		HealthCheckFilterConfig: HealthCheckFilterConfig{
			PassThrough: true,
			Endpoint:    "test",
			ClusterMinHealthyPercentage: map[string]float32{
				"test": 10.0,
			},
		},
		CacheTime: time.Second,
	}
	b, err := json.Marshal(hc)
	if err != nil {
		t.Fatal(err)
	}
	nhc := &HealthCheckFilter{}
	if err := json.Unmarshal(b, nhc); err != nil {
		t.Fatal(err)
	}
	if !(hc.PassThrough == nhc.PassThrough &&
		hc.Endpoint == nhc.Endpoint &&
		len(hc.ClusterMinHealthyPercentage) == len(nhc.ClusterMinHealthyPercentage) &&
		hc.ClusterMinHealthyPercentage["test"] == nhc.ClusterMinHealthyPercentage["test"] &&
		hc.CacheTime == nhc.CacheTime) {
		t.Error("unmarshal result is not expected")
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

func TestRouterMarshal(t *testing.T) {
	router := &Router{
		Metadata: map[string]string{
			"label": "gray",
		},
	}
	b, err := json.Marshal(router)
	if err != nil {
		t.Fatal(err)
	}
	nrouter := &Router{}
	if err := json.Unmarshal(b, nrouter); err != nil {
		t.Fatal(err)
	}
	if v, ok := nrouter.Metadata["label"]; !ok || v != "gray" {
		t.Fatal("unmarshal result is not expected")
	}
}

func TestRouterUnmarshal(t *testing.T) {
	cfgStr := `{
			"metadata": {
				"filter_metadata": {
					"mosn.lb": {
						"label": "gray"
					}
				}
			}
		}`
	router := &Router{}
	if err := json.Unmarshal([]byte(cfgStr), router); err != nil {
		t.Fatal(err)
	}
	if v, ok := router.Metadata["label"]; !ok || v != "gray" {
		t.Fatal("unmarshal result is not expected")
	}
}

func TestRouterActionMarshal(t *testing.T) {
	routerAction := &RouteAction{
		MetadataMatch: api.Metadata{
			"label": "gray",
		},
		Timeout: time.Second,
	}
	b, err := json.Marshal(routerAction)
	if err != nil {
		t.Fatal(err)
	}
	nra := &RouteAction{}
	if err := json.Unmarshal(b, nra); err != nil {
		t.Fatal(err)
	}
	if !(len(nra.MetadataMatch) == 1 &&
		nra.MetadataMatch["label"] == "gray" &&
		nra.Timeout == time.Second) {
		t.Error("unmarshal and marshal is not equal")
	}
}

func TestRouterActionUnmarshal(t *testing.T) {
	cfgStr := `{
		"cluster_name": "test",
		"weighted_clusters": [
			{
				"cluster": {
					"name": "test"
				}
			}
		],
		"metadata_match": {
			"filter_metadata": {
				"mosn.lb": {
					"label": "gray"
				}
			}
		},
		"timeout": "1s",
		"retry_policy": {
			"retry_on": true,
			"retry_timeout": "1s"
		},
		"request_headers_to_add": [
			{
				"header": {
					"key": "test",
					"value": "ok"
				}
			}
		],
		"request_headers_to_remove":["test"]
	}`
	routerAction := &RouteAction{}
	if err := json.Unmarshal([]byte(cfgStr), routerAction); err != nil {
		t.Fatal(err)
	}
	if !(routerAction.ClusterName == "test" &&
		len(routerAction.WeightedClusters) == 1 &&
		routerAction.WeightedClusters[0].Cluster.Name == "test" &&
		len(routerAction.MetadataMatch) == 1 &&
		routerAction.MetadataMatch["label"] == "gray" &&
		routerAction.Timeout == time.Second &&
		routerAction.RetryPolicy.RetryOn &&
		routerAction.RetryPolicy.RetryTimeout == time.Second &&
		len(routerAction.RequestHeadersToAdd) == 1 &&
		routerAction.RequestHeadersToAdd[0].Header.Key == "test" &&
		routerAction.RequestHeadersToAdd[0].Header.Value == "ok" &&
		len(routerAction.RequestHeadersToRemove) == 1 &&
		routerAction.RequestHeadersToRemove[0] == "test") {

		t.Errorf("unmarshal is not expected, %v", routerAction)
	}
}

func TestClusterWeightMarshal(t *testing.T) {
	cw := &ClusterWeight{
		MetadataMatch: api.Metadata{
			"label": "gray",
		},
	}
	b, err := json.Marshal(cw)
	if err != nil {
		t.Fatal(err)
	}
	ncw := &ClusterWeight{}
	if err := json.Unmarshal(b, ncw); err != nil {
		t.Fatal(err)
	}
	if !(len(ncw.MetadataMatch) == 1 &&
		ncw.MetadataMatch["label"] == "gray") {
		t.Error("unmarshal and marshal is not equal")
	}
}

func TestClusterWeightUnmarshal(t *testing.T) {
	cfgStr := `{
		"name": "test",
		"weight": 100,
		"metadata_match": {
			"filter_metadata": {
				"mosn.lb": {
					"label": "gray"
				}
			}
		}
	}`
	cw := &ClusterWeight{}
	if err := json.Unmarshal([]byte(cfgStr), cw); err != nil {
		t.Fatal(err)
	}
	if !(cw.Name == "test" &&
		cw.Weight == 100 &&
		len(cw.MetadataMatch) == 1 &&
		cw.MetadataMatch["label"] == "gray") {
		t.Errorf("unmarshal unepxetced, %v", cw)
	}
}

func TestRetryPolicyMarshal(t *testing.T) {
	p := &RetryPolicy{
		RetryPolicyConfig: RetryPolicyConfig{
			RetryOn:    true,
			NumRetries: 3,
		},
		RetryTimeout: time.Second,
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	np := &RetryPolicy{}
	if err := json.Unmarshal(b, np); err != nil {
		t.Fatal(err)
	}
	if !(np.RetryOn &&
		np.NumRetries == 3 &&
		np.RetryTimeout == time.Second) {
		t.Error("marshal and unmarshal not equal")
	}
}

func TestRetryPolicyUnmarshal(t *testing.T) {
	cfgStr := `{
		"retry_on": true,
		"retry_timeout": "1s",
		"num_retries": 3
	}`
	p := &RetryPolicy{}
	if err := json.Unmarshal([]byte(cfgStr), p); err != nil {
		t.Fatal(err)
	}
	if !(p.RetryOn &&
		p.NumRetries == 3 &&
		p.RetryTimeout == time.Second) {
		t.Errorf("unmarshal unexpected %v", p)
	}
}

func TestCircuitBreakersMarshal(t *testing.T) {
	cb := &CircuitBreakers{
		Thresholds: []Thresholds{
			{
				MaxConnections: 1024,
			},
		},
	}
	b, err := json.Marshal(cb)
	if err != nil {
		t.Fatal(err)
	}
	ncb := &CircuitBreakers{}
	if err := json.Unmarshal(b, ncb); err != nil {
		t.Fatal(err)
	}
	if !(len(ncb.Thresholds) == 1 &&
		ncb.Thresholds[0].MaxConnections == 1024) {
		t.Error("marshal and unmarshal not equal")
	}
}

func TestCircuitBreakersUnmarshal(t *testing.T) {
	cfgStr := `[
		{
			"priority": "DEFAULT",
			"max_connections": 1024,
			"max_requests": 1024
		}
	]`
	cb := &CircuitBreakers{}
	if err := json.Unmarshal([]byte(cfgStr), cb); err != nil {
		t.Fatal(err)
	}
	if !(len(cb.Thresholds) == 1 &&
		cb.Thresholds[0].MaxConnections == 1024 &&
		cb.Thresholds[0].MaxRequests == 1024) {
		t.Errorf("unmarshal unexpected %v", cb)
	}
}

func TestFilterChainMarshal(t *testing.T) {
	filterChain := &FilterChain{
		TLSContexts: []TLSConfig{
			{
				Status: true,
			},
		},
	}
	b, err := json.Marshal(filterChain)
	if err != nil {
		t.Fatal("marshal filter chain error: ", err)
	}
	expectedStr := `{"tls_context_set":[{"status":true}]}`
	if string(b) != expectedStr {
		t.Error("marshal filter chain unexpected, got: ", string(b))
	}
}

func TestFilterChainUnmarshal(t *testing.T) {
	defaultTLS := `{
		"match": "test_default",
		"filters": [
			{
				"type": "proxy"
			}
		]
	}`
	singleTLS := `{
		"match": "test_single",
		"tls_context": {
			"status": true
		},
		"filters": [
			{
				"type": "proxy"
			}
		]
	}`
	multiTLS := `{
		"match": "test_multi",
		"tls_context_set": [
			{
				"status": true
			},
			{
				"status": true
			}
		],
		"filters": [
			{
				"type": "proxy"
			}
		]
	}`
	defaultChain := &FilterChain{}
	if err := json.Unmarshal([]byte(defaultTLS), defaultChain); err != nil {
		t.Fatalf("unmarshal default tls config error: %v", err)
	}
	if len(defaultChain.TLSContexts) != 1 || defaultChain.TLSContexts[0].Status {
		t.Fatalf("unmarshal tls context unexpected")
	}
	for i, cfgStr := range []string{singleTLS, multiTLS} {
		filterChain := &FilterChain{}
		if err := json.Unmarshal([]byte(cfgStr), filterChain); err != nil {
			t.Errorf("#%d unmarshal error: %v", i, err)
			continue
		}
		if len(filterChain.TLSContexts) < 1 {
			t.Errorf("#%d tls contexts unmarshal not expected, got %v", i, filterChain)
		}
		for _, ctx := range filterChain.TLSContexts {
			if !ctx.Status {
				t.Errorf("#%d tls contexts unmarshal failed", i)
			}
		}
	}
	// expected an error
	duplicateTLS := `{
		"match": "test_multi",
		"tls_context": {
			"status": true
		},
		"tls_context_set": [
			{
				"status": true
			}
		],
		"filters": [
			{
				"type": "proxy"
			}
		]
	}
	`
	errCompare := func(e error) bool {
		if e == nil {
			return false
		}
		return strings.Contains(e.Error(), ErrDuplicateTLSConfig.Error())
	}
	filterChain := &FilterChain{}
	if err := json.Unmarshal([]byte(duplicateTLS), filterChain); !errCompare(err) {
		t.Errorf("expected a duplicate error, but not, got: %v", err)
	}
}

func TestRouterConfigMarshal(t *testing.T) {
	router := &RouterConfiguration{
		VirtualHosts: []VirtualHost{
			{
				Name:    "test",
				Domains: []string{"*"},
			},
		},
	}
	b, err := json.Marshal(router)
	if err != nil {
		t.Fatal(err)
	}
	nrouter := &RouterConfiguration{}
	if err := json.Unmarshal(b, nrouter); err != nil {
		t.Fatal(err)
	}
	if !(len(nrouter.VirtualHosts) == 1 &&
		nrouter.VirtualHosts[0].Name == "test" &&
		len(nrouter.VirtualHosts[0].Domains) == 1 &&
		router.VirtualHosts[0].Domains[0] == "*") {
		t.Error("unmarshal and marshal is not equal")
	}
}
func TestRouterConfigUmarshal(t *testing.T) {
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
							],
							"variables":[
								{
									"name":"header",
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
			vh.Domains[0] == "*") {
			t.Error("virtual host failed")
		}
		if len(vh.Routers) != 1 {
			t.Error("virtual host failed")
		} else {
			router := vh.Routers[0]
			if !(router.Match.Prefix == "/" &&
				len(router.Match.Headers) == 1 &&
				router.Match.Headers[0].Name == "service" &&
				router.Match.Headers[0].Value == "test" &&
				router.Match.Variables[0].Name == "header" &&
				router.Match.Variables[0].Value == "test") {
				t.Error("virtual host failed")
			}
			meta := api.Metadata{
				"test": "test",
			}
			if !(router.Route.ClusterName == "cluster" &&
				router.Route.Timeout == time.Second &&
				router.Route.RetryPolicy.RetryTimeout == time.Minute &&
				router.Route.RetryPolicy.RetryOn == true &&
				router.Route.RetryPolicy.NumRetries == 10 &&
				reflect.DeepEqual(meta, router.Metadata)) {
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

		}
	}

}
func TestRouterConfigConflict(t *testing.T) {
	routerConfig := `{
		"router_config_name":"test_router",
		"router_configs":"/tmp/routers/test_routers/",
		"virtual_hosts": [
			{
				"name": "virtualhost"
			}
		]
	}`
	errCompare := func(e error) bool {
		if e == nil {
			return false
		}
		return strings.Contains(e.Error(), ErrDuplicateStaticAndDynamic.Error())
	}
	if err := json.Unmarshal([]byte(routerConfig), &RouterConfiguration{}); !errCompare(err) {
		t.Fatalf("test config conflict with both dynamic mode and static mode failed, get error: %v", err)
	}
}

func TestRouterMarshalWithSep(t *testing.T) {
	routerName := "router_config_name"
	routerPath := path.Join("/tmp/routers_path", routerName)
	vhWithSep := "test/vh/with/sep"
	os.RemoveAll(routerPath)
	rcfg := &RouterConfiguration{
		VirtualHosts: []VirtualHost{
			VirtualHost{
				Name:    vhWithSep,
				Domains: []string{"*"},
			},
		},
		RouterConfigurationConfig: RouterConfigurationConfig{
			RouterConfigName: routerName,
			RouterConfigPath: routerPath,
		},
	}
	if _, err := json.Marshal(rcfg); err != nil {
		t.Fatal(err)
	}
	// verify
	data, err := ioutil.ReadFile(path.Join(routerPath, "test_vh_with_sep.json"))
	if err != nil || !strings.Contains(string(data), vhWithSep) {
		t.Fatalf("read router file failed, error: %v, data: %s", err, string(data))
	}

}

func TestRouterConfigDynamicModeParse(t *testing.T) {
	routerPath := "/tmp/routers/test_routers"
	os.RemoveAll(routerPath)
	if err := os.MkdirAll(routerPath, 0755); err != nil {
		t.Fatal(err)
	}
	// dynamic mode
	// write some files
	virtualHostConfigs := []string{
		`{
			"name": "virtualhost_0"
		}`,
		`{
			"name": "virtualhost_1"
		}`,
	}
	for i, vh := range virtualHostConfigs {
		data := []byte(vh)
		fileName := fmt.Sprintf("%s/virtualhost_%d.json", routerPath, i)
		if err := ioutil.WriteFile(fileName, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// write ignore error file
	for _, f := range []struct {
		fileName string
		data     []byte
	}{
		{
			fileName: fmt.Sprintf("%s/virtualhost_notjson.file", routerPath),
			data:     []byte("12345"),
		},
		{
			fileName: fmt.Sprintf("%s/virtualhost_empty.json", routerPath),
		},
	} {
		if err := ioutil.WriteFile(f.fileName, f.data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// read dynamic mode config
	routerConfig := `{
		"router_config_name":"test_router",
		"router_configs":"/tmp/routers/test_routers/"
	}`
	testConfig := &RouterConfiguration{}
	if err := json.Unmarshal([]byte(routerConfig), testConfig); err != nil {
		t.Fatal(err)
	}
	// verify
	if len(testConfig.VirtualHosts) != 2 {
		t.Fatalf("virtual host parsed not enough, got: %v", testConfig.VirtualHosts)
	}
	// add a new virtualhost
	testConfig.VirtualHosts = append(testConfig.VirtualHosts, VirtualHost{
		Domains: []string{"*"},
	})
	// dump json
	if _, err := json.Marshal(testConfig); err != nil {
		t.Fatal(err)
	}
	// verify
	files, err := ioutil.ReadDir(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("new virtual host is not dumped, just got %d files", len(files))
	}
	// test delete virtualhost
	testConfig.VirtualHosts = testConfig.VirtualHosts[:1]
	// dump json
	if _, err := json.Marshal(testConfig); err != nil {
		t.Fatal(err)
	}
	// verify
	files, err = ioutil.ReadDir(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("new virtual host is not dumped, just got %d files", len(files))
	}
}

func TestListenerMarshal(t *testing.T) {
	addrStr := "0.0.0.0:8080"
	addr, err := net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		t.Fatal(err)
	}
	ln := &Listener{
		ListenerConfig: ListenerConfig{
			Name:       "test_listener",
			Type:       INGRESS,
			BindToPort: true,
			Inspector:  true,
		},
		Addr: addr,
	}
	b, err := json.Marshal(ln)
	if err != nil {
		t.Fatal(err)
	}
	ln2 := &Listener{}
	if err := json.Unmarshal(b, ln2); err != nil {
		t.Fatal(err)
	}
	if !(ln2.AddrConfig == addrStr && // addr config will be replaced by net.Addr
		ln2.Name == "test_listener" &&
		ln2.Type == INGRESS &&
		ln2.Inspector == true &&
		ln2.ConnectionIdleTimeout == nil) {
		t.Fatalf("listener config marshal unepxected, got :%v", ln2)
	}
}

func TestHashPolicyUnmarshal(t *testing.T) {
	config := `{
		"hash_policy": [{
			"header": {"key":"header_key"}
		}]
	}`

	headerConfig := &RouterActionConfig{}
	err := json.Unmarshal([]byte(config), headerConfig)
	if !assert.NoErrorf(t, err, "error should be nil, get %+v", err) {
		t.FailNow()
	}
	if !assert.NotNilf(t, headerConfig.HashPolicy[0].Header,
		"header should not be nil") {
		t.FailNow()
	}
	header := headerConfig.HashPolicy[0].Header.Key
	if !assert.Equalf(t, "header_key", header,
		"header key should be header_key, get %s", header) {
		t.FailNow()
	}

	config2 := `{
		"hash_policy": [{
			"cookie": {
				"name": "name",
				"path": "path",
				"ttl": "5s"
			}
		}]
	}`

	cookieConfig := &RouterActionConfig{}
	err = json.Unmarshal([]byte(config2), cookieConfig)
	if !assert.NoErrorf(t, err, "error should be nil, get %+v", err) {
		t.FailNow()
	}
	if !assert.NotNilf(t, cookieConfig.HashPolicy[0].Cookie,
		"HttpCookie should not be nil") {
		t.FailNow()
	}
	name := cookieConfig.HashPolicy[0].Cookie.Name
	path := cookieConfig.HashPolicy[0].Cookie.Path
	ttl := cookieConfig.HashPolicy[0].Cookie.TTL.Duration
	if !assert.Equalf(t, "name", name, "cookie key should be name, get %s", name) {
		t.FailNow()
	}
	if !assert.Equalf(t, "path", path, "cookie path should be path, get %s", path) {
		t.FailNow()
	}
	if !assert.Equalf(t, 5*time.Second, ttl, "cookie ttl should be 5s, get %s", ttl) {
		t.FailNow()
	}

	config3 := `{
		"hash_policy": [{
			"source_ip":{}
		}]
	}`

	sourceIPConfig := &RouterActionConfig{}
	err = json.Unmarshal([]byte(config3), sourceIPConfig)
	if !assert.NoErrorf(t, err, "error should be nil, get %+v", err) {
		t.FailNow()
	}
	if !assert.NotNilf(t, sourceIPConfig.HashPolicy[0].SourceIP,
		"SourceIP should not be nil") {
		t.FailNow()
	}
}

func TestHashPolicyMarshal(t *testing.T) {
	config := `{"hash_policy":[{"header":{"key":"header_key"}}],"timeout":"0s"}`

	headerConfig := &RouterActionConfig{
		HashPolicy: []HashPolicy{
			{
				Header: &HeaderHashPolicy{Key: "header_key"},
			},
		},
	}
	b, err := json.Marshal(headerConfig)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !assert.Equalf(t, config, string(b), "marshal hash policy expect to get %s, but get %s", config, string(b)) {
		t.FailNow()
	}

	config2 := `{"hash_policy":[{"cookie":{"name":"name","path":"path","ttl":"5s"}}],"timeout":"0s"}`
	cookieConfig := &RouterActionConfig{
		HashPolicy: []HashPolicy{
			{
				Cookie: &CookieHashPolicy{
					Name: "name",
					Path: "path",
					TTL:  api.DurationConfig{5 * time.Second},
				},
			},
		},
	}
	b, err = json.Marshal(cookieConfig)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !assert.Equalf(t, config2, string(b), "marshal hash policy expect to get %s, but get %s", config2, string(b)) {
		t.FailNow()
	}

	config3 := `{"hash_policy":[{"source_ip":{}}],"timeout":"0s"}`
	ipConfig := &RouterActionConfig{
		HashPolicy: []HashPolicy{
			{
				SourceIP: &SourceIPHashPolicy{},
			},
		},
	}
	b, err = json.Marshal(ipConfig)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !assert.Equalf(t, config3, string(b), "marshal hash policy expect to get %s, but get %s", config3, string(b)) {
		t.FailNow()
	}
}
