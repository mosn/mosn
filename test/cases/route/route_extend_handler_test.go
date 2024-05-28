//go:build MOSNTest
// +build MOSNTest

package route

import (
	"fmt"
	"testing"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/http"
)

func TestRouteExtendHandler(t *testing.T) {
	Scenario(t, "mosn route handler with check, no host cluster will be ignore", func() {
		ConfigExtendRoute := fmt.Sprintf(ConfigExtendRouteTmpl, "")
		_, _ = lib.InitMosn(ConfigExtendRoute, lib.CreateConfig(MockHttpServerConfig))
		Case("route extend", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Request: &http.RequestConfig{
					Header: map[string][]string{
						"x-test-route": []string{"paas"},
						"x-first":      []string{"test"},
					},
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200, // x-first contains no host, fallback to x-test-route
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("route only matched", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Request: &http.RequestConfig{
					Header: map[string][]string{
						"x-first": []string{"test"},
					},
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 404, // no route matched
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("route default", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Request: &http.RequestConfig{
					Header: map[string][]string{
						"x-test-route": []string{"paas"},
						"x-first":      []string{"test"},
					},
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 502, // choose a cluster, but no hosts
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
	})
}

func TestRouteExtendHandler2(t *testing.T) {
	Scenario(t, "mosn route handler withcheck, ", func() {
		ConfigExtendRoute := fmt.Sprintf(ConfigExtendRouteTmpl, `{"address":"127.0.0.1:8081"}`) // a host that no listened
		_, _ = lib.InitMosn(ConfigExtendRoute, lib.CreateConfig(MockHttpServerConfig))
		Case("route extend", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Request: &http.RequestConfig{
					Header: map[string][]string{
						"x-test-route": []string{"paas"},
						"x-first":      []string{"test"},
					},
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 502, // connection failed
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("route default", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Request: &http.RequestConfig{
					Header: map[string][]string{
						"x-test-route": []string{"paas"},
						"x-first":      []string{"test"},
					},
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 502,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
	})
}

const ConfigExtendRouteTmpl = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "ERROR",
                        "routers": [
                                {
                                        "router_config_name":"router_to_server",
                                        "virtual_hosts":[{
                                                "name":"server_hosts",
                                                "domains": ["*"],
                                                "routers": [
							{
								"match":{"headers":[{"name":"x-first","value":"test"}]},
								"route":{"cluster_name":"priority_cluster"}
							},
                                                        {
                                                                "match":{"headers":[{"name":"x-test-route","value":"paas"}]},
                                                                "route":{"cluster_name":"server_cluster"}
                                                        }
                                                ]
                                        }]
                                }
                        ],
                        "listeners":[
                                {
                                        "address":"127.0.0.1:2046",
                                        "bind_port": true,
                                        "filter_chains": [{
                                                "filters": [
                                                        {
                                                                "type": "proxy",
                                                                "config": {
                                                                        "downstream_protocol": "Http1",
                                                                        "upstream_protocol": "Http1",
                                                                        "router_config_name":"router_to_server",
									"router_handler_name": "check-handler"
                                                                }
                                                        }
                                                ]
                                        }]
                                },
				{
					"address":"127.0.0.1:2045",
					"bind_port": true,
					"filter_chains": [{
						"filters": [
							{
								"type": "proxy",
								"config": {
									"downstream_protocol": "Http1",
									"upstream_protocol": "Http1",
									"router_config_name":"router_to_server"
								}
							}
						]
					}]
				}
                        ]
                }
        ],
        "cluster_manager":{
                "clusters":[
                        {
                                "name": "server_cluster",
                                "type": "SIMPLE",
                                "lb_type": "LB_RANDOM",
                                "hosts":[
                                        {"address":"127.0.0.1:8080"}
                                ]
                        },
			{
				"name": "priority_cluster",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
				"hosts":[
					%s
				]
			}
                ]
        }
}`
