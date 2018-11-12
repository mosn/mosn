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
	"istio.io/api/mixer/v1"
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

var mockedFilterChains = `
            {
             "match":"",
             "tls_context":{
               "status": true,
               "inspector": true,
               "server_name": "hello.com",
               "ca_cert": "-----BEGIN CERTIFICATE-----\nMIIDMjCCAhoCCQDaFC8PcSS5qTANBgkqhkiG9w0BAQsFADBbMQswCQYDVQQGEwJD\nTjEKMAgGA1UECAwBYTEKMAgGA1UEBwwBYTEKMAgGA1UECgwBYTEKMAgGA1UECwwB\nYTEKMAgGA1UEAwwBYTEQMA4GCSqGSIb3DQEJARYBYTAeFw0xODA2MTQwMjQyMjVa\nFw0xOTA2MTQwMjQyMjVaMFsxCzAJBgNVBAYTAkNOMQowCAYDVQQIDAFhMQowCAYD\nVQQHDAFhMQowCAYDVQQKDAFhMQowCAYDVQQLDAFhMQowCAYDVQQDDAFhMRAwDgYJ\nKoZIhvcNAQkBFgFhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArbNc\nmOXvvOqZgJdxMIiklE8lykVvz5d7QZ0+LDVG8phshq9/woigfB1aFBAI36/S5LZQ\n5Fd0znblSa+LOY06jdHTkbIBYFlxH4tdRaD0B7DbFzR5bpzLv2Q+Zf5u5RI73Nky\nH8CjW9QJjboArHkwm0YNeENaoR/96nYillgYLnunol4h0pxY7ZC6PpaB1EBaTXcz\n0iIUX4ktUJQmYZ/DFzB0oQl9IWOj18ml2wYzu9rYsySzj7EPnDOOebsRfS5hl3fz\nHi4TC4PDh0mQwHqDQ4ncztkybuRSXFQ6RzEPdR5qtp9NN/G/TlfyB0CET3AFmGkp\nE2irGoF/JoZXEDeXmQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQApzhQLS7fAcExZ\nx1S+hcy7lLF8QcPlsiH32SnLFg5LPy4prz71mebUchmt97t4T3tSWzwXi8job7Q2\nONYc6sr1LvaFtg7qoCfz5fPP5x+kKDkEPwCDJSTVPcXP+UtA407pxX8KPRN8Roay\ne3oGcmNqVu/DkkufkIL3PBg41JEMovWtKD+PXmeBafc4vGCHSJHJBmzMe5QtwHA0\nss/A9LHPaq3aLcIyFr8x7clxc7zZVaim+lVfNV3oPBnB4gU7kLFVT0zOhkM+V1A4\nQ5GVbGAu4R7ItY8kJ2b7slre0ajPUp2FMregt4mnUM3mu1nbltVhtoknXqHHMGgN\n4Lh4JfNx\n-----END CERTIFICATE-----\n",
               "cert_chain": "-----BEGIN CERTIFICATE-----\nMIIDJTCCAg0CAQEwDQYJKoZIhvcNAQELBQAwWzELMAkGA1UEBhMCQ04xCjAIBgNV\nBAgMAWExCjAIBgNVBAcMAWExCjAIBgNVBAoMAWExCjAIBgNVBAsMAWExCjAIBgNV\nBAMMAWExEDAOBgkqhkiG9w0BCQEWAWEwHhcNMTgwNjE0MDMxMzQyWhcNMTkwNjE0\nMDMxMzQyWjBWMQswCQYDVQQGEwJDTjEKMAgGA1UECAwBYTEKMAgGA1UECgwBYTEK\nMAgGA1UECwwBYTERMA8GA1UEAwwIdGVzdC5jb20xEDAOBgkqhkiG9w0BCQEWAWEw\nggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCrPq+Mo0nS3dJU1qGFwlIB\ni9HqRm5RGcfps+0W5LjEhqUKxKUweRrwDaIxpiSqjKeehz9DtLUpXBD29pHuxODU\nVsMH2U1AGWn9l4jMnP6G5iTMPJ3ZTXszeqALe8lm/f807ZA0C7moc+t7/d3+b6d2\nlnwR+yWbIZJUu2qw+HrR0qPpNlBP3EMtlQBOqf4kCl6TfpqrGfc9lW0JjuE6Taq3\ngSIIgzCsoUFe30Yemho/Pp4zA9US97DZjScQLQAGiTsCRDBASxXGzODQOfZL3bCs\n2w//1lqGjmhp+tU1nR4MRN+euyNX42ioEz111iB8y0VzuTIsFBWwRTKK1SF7YSEb\nAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABnRM9JJ21ZaujOTunONyVLHtmxUmrdr\n74OJW8xlXYEMFu57Wi40+4UoeEIUXHviBnONEfcITJITYUdqve2JjQsH2Qw3iBUr\nmsFrWS25t/Krk2FS2cKg8B9azW2+p1mBNm/FneMv2DMWHReGW0cBp3YncWD7OwQL\n9NcYfXfgBgHdhykctEQ97SgLHDKUCU8cPJv14eZ+ehIPiv8cDWw0mMdMeVK9q71Y\nWn2EgP7HzVgdbj17nP9JJjNvets39gD8bU0g2Lw3wuyb/j7CHPBBzqxh+a8pihI5\n3dRRchuVeMQkMuukyR+/A8UrBMA/gCTkXIcP6jKO1SkKq5ZwlMmapPc=\n-----END CERTIFICATE-----\n",
               "private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAqz6vjKNJ0t3SVNahhcJSAYvR6kZuURnH6bPtFuS4xIalCsSl\nMHka8A2iMaYkqoynnoc/Q7S1KVwQ9vaR7sTg1FbDB9lNQBlp/ZeIzJz+huYkzDyd\n2U17M3qgC3vJZv3/NO2QNAu5qHPre/3d/m+ndpZ8EfslmyGSVLtqsPh60dKj6TZQ\nT9xDLZUATqn+JApek36aqxn3PZVtCY7hOk2qt4EiCIMwrKFBXt9GHpoaPz6eMwPV\nEvew2Y0nEC0ABok7AkQwQEsVxszg0Dn2S92wrNsP/9Zaho5oafrVNZ0eDETfnrsj\nV+NoqBM9ddYgfMtFc7kyLBQVsEUyitUhe2EhGwIDAQABAoIBAG2Bj5ca0Fmk+hzA\nh9fWdMSCWgE7es4n81wyb/nE15btF1t0dsIxn5VE0qR3P1lEyueoSz+LrpG9Syfy\nc03B3phKxzscrbbAybOeFJ/sASPYxk1IshRE5PT9hJzzUs6mvG1nQWDW4qmjP0Iy\nDKTpV6iRANQqy1iRtlay5r42l6vWwHfRfwAv4ExSS+RgkYcavqOp3e9If2JnFJuo\n7Zds2i7Ux8dURX7lHqKxTt6phgoMmMpvO3lFYVGos7F13OR9NKElzjiefAQbweAt\nt8R+6A1rlIwnfywxET9ZXglfOFK6Q0nqCJhcEcKzT/Xfkd+h9XPACjOObJh3a2+o\nwg9GBFECgYEA2a6JYuFanKzvajFPbSeN1csfI9jPpK2+tB5+BB72dE74B4rjygiG\n0Rb26UjovkYfJJqKuKr4zDL5ziSlJk199Ae2f6T7t7zmyhMlWQtVT12iTQvBINTz\nNerKi5HNVBsCSGj0snbwo8u4QRgTjaIoVqTlOlUQuGqYuZ75l8g35IkCgYEAyWOM\nKagzpGmHWq/0ThN4kkwWOdujxuqrPf4un2WXsir+L90UV7X9wY4mO19pe5Ga2Upu\nXFDsxAZsanf8SbzkTGHvzUobFL7eqsiwaUSGB/cGEtkIyVlAdyW9DhiZFt3i9mEF\nbBsHnEDHPHL4tu+BB8G3WahHjWOnbWZ3NTtP94MCgYEAi3XRmSLtjYER5cPvsevs\nZ7M5oRqvdT7G9divPW6k0MEjEJn/9BjgXqbKy4ylZ/m+zBGinEsVGKXz+wjpMY/m\nCOjEGCUYC5AfgAkiHVkwb6d6asgEFEe6BaoF18MyfBbNsJxlYMzowNeslS+an1vr\nYg9EuMl06+GHNSzPlVl1zZkCgYEAxXx8N2F9eu4NUK4ZafMIGpbIeOZdHbSERp+b\nAq5yasJkT3WB/F04QXVvImv3Gbj4W7r0rEyjUbtm16Vf3sOAMTMdIHhaRCbEXj+9\nVw1eTjM8XoE8b465e92jHk6a2WSvq6IK2i9LcDvJ5QptwZ7uLjgV37L4r7sYtVx0\n69uFGJcCgYEAot7im+Yi7nNsh1dJsDI48liKDjC6rbZoCeF7Tslp8Lt+4J9CA9Ux\nSHyKjg9ujbjkzCWrPU9hkugOidDOmu7tJAxB5cS00qJLRqB5lcPxjOWcBXCdpBkO\n0tdT/xRY/MYLf3wbT95enaPlhfeqBBXKNQDya6nISbfwbMLfNxdZPJ8=\n-----END RSA PRIVATE KEY-----\n",
               "verify_client": true,
               "cipher_suites": "ECDHE-RSA-AES256-GCM-SHA384",
               "ecdh_curves": "P256"
               },
               "filters": [
                {
                  "type": "proxy",
                  "config": {
                    "downstream_protocol": "SofaRpc",
                    "name": "proxy_config",
                    "upstream_protocol": "SofaRpc",
                    "router_config_name":"test_router"
                  }
                },
                {
                  "type":"connection_manager",
                  "config":{
                    "router_config_name":"test_router",
                    "virtual_hosts": [
                      {
                        "name": "sofa",
                        "require_tls": "no",
                        "domains":[
                          "*testwilccard"
                        ],
                        "routers": [
                          {
                            "match": {
                              "headers": [
                                {
                                  "name": "service",
                                  "value": "com.alipay.rpc.common.service.facade.pb.SampleServicePb:1.0",
                                  "regex":false
                                }
                              ]
                            },
                            "route": {
                              "cluster_name": "test_cpp",
                              "metadata_match": {
                                "filter_metadata": {
                                  "mosn.lb": {
                                    "version":"1.1",
                                    "stage":"pre-release",
                                    "label": "gray"
                                  }
                                }
                              },
                              "weighted_clusters":[
                                {
                                  "cluster":{
                                    "name":"serverCluster1",
                                    "weight":90,m
                                    "metadata_match":{
                                      "filter_metadata": {
                                        "mosn.lb": {
                                          "version": "v1"
                                        }
                                      }
                                    }
                                  }
                                },
                                {
                                  "cluster":{
                                    "name":"serverCluster2",
                                    "weight":10,
                                    "metadata_match":{
                                      "filter_metadata": {
                                        "mosn.lb": {
                                          "version": "v2"
                                        }
                                      }
                                    }
                                  }
                                }
                              ]
                            }
                          }
                        ]
                      }
                    ]
                  }
                }
              ]
            }`

func TestParseRouterConfiguration(t *testing.T) {
	bytes := []byte(mockedFilterChains)
	filterChan := &v2.FilterChain{}
	json.Unmarshal(bytes, filterChan)

	routerCfg := ParseRouterConfiguration(filterChan)

	if routerCfg.RouterConfigName != "test_router" || len(routerCfg.VirtualHosts) != 1 ||
		routerCfg.VirtualHosts[0].Name != "sofa" || len(routerCfg.VirtualHosts[0].Routers) != 1 {
		t.Errorf("TestParseRouterConfiguration error")
	}
}

func TestGetListenerDisableIO(t *testing.T) {
	bytes := []byte(mockedFilterChains)
	filterChan := &v2.FilterChain{}
	json.Unmarshal(bytes, filterChan)

	wanted := false

	if disableIO := GetListenerDisableIO(filterChan); disableIO != wanted {
		t.Errorf("TestGetListenerDisableIO error, want %t but got %t ", disableIO, wanted)
	}
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

	if c.MaxRequestPerConn != DefaultMaxRequestPerConn {
		t.Errorf("Expect cluster.MaxRequestPerConn default value %d but got %d",
			DefaultMaxRequestPerConn, c.MaxRequestPerConn)
	}

	if c.ConnBufferLimitBytes != DefaultConnBufferLimitBytes {
		t.Errorf("Expect cluster.ConnBufferLimitBytes default value%d, but got %d",
			DefaultConnBufferLimitBytes, c.ConnBufferLimitBytes)
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
			LogLevelConfig: "DEBUG",
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
		"downstream_protocol": "SofaRpc",
		"upstream_protocol": "Http2",
		"router_config_name":"test_router",
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
		proxy.UpstreamProtocol == string(protocol.HTTP2) && proxy.RouterConfigName == "test_router") {
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
		"stat_prefix":          "tcp_proxy",
		"cluster":              "cluster",
		"max_connect_attempts": 1000,
		"routes": []interface{}{
			map[string]interface{}{
				"cluster": "test",
				"SourceAddrs": []interface{}{
					map[string]interface{}{
						"address": "127.0.0.1",
						"length":  32,
					},
				},
				"DestinationAddrs": []interface{}{
					map[string]interface{}{
						"address": "127.0.0.1",
						"length":  32,
					},
				},
				"SourcePort":      "8080",
				"DestinationPort": "8080",
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
func TestParseServiceRegistry(t *testing.T) {
	cb.Count = 0
	ParseServiceRegistry(v2.ServiceRegistryInfo{})
	if cb.Count != 1 {
		t.Error("no callback")
	}
}

func TestParseMixerFilter(t *testing.T) {
	m := map[string]interface{}{
		"mixer_attributes": map[string]interface{}{
			"attributes": map[string]interface{}{
				"context.reporter.kind": map[string]interface{}{
					"string_value": "outbound",
				},
			},
		},
	}

	mixer := ParseMixerFilter(m)
	if mixer == nil {
		t.Errorf("parse mixer config error")
	}

	if mixer.MixerAttributes == nil {
		t.Errorf("parse mixer config error")
	}
	val, exist := mixer.MixerAttributes.Attributes["context.reporter.kind"]
	if !exist {
		t.Errorf("parse mixer config error")
	}

	strVal, ok := val.Value.(*v1.Attributes_AttributeValue_StringValue)
	if !ok {
		t.Errorf("parse mixer config error")
	}
	if strVal.StringValue != "outbound" {
		t.Errorf("parse mixer config error")
	}
}
