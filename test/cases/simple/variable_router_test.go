//go:build MOSNTest
// +build MOSNTest

package simple

import (
	"encoding/json"
	"testing"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/http"
)

func TestVariableRoute1(t *testing.T) {
	Scenario(t, "variable route test", func() {
		// servers is invalid in `Case`
		_, servers := lib.InitMosn(VariableRouterConfigHTTP1, lib.CreateConfig(genMockHttpServerConfig("server1", "127.0.0.1:8080")),
			lib.CreateConfig(genMockHttpServerConfig("server2", "127.0.0.1:8081")))
		Case("client-mosn-mosn-server1", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedHeader: map[string][]string{
						"server_address": []string{"127.0.0.1:8080"},
					},
					ExpectedBody: []byte("default-http1"),
				},
				Request: &http.RequestConfig{
					Method: "GET",
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)
		})
		Case("client-mosn-mosn-server2", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedHeader: map[string][]string{
						"server_address": []string{"127.0.0.1:8081"},
					},
					ExpectedBody: []byte("default-http1"),
				},
				Request: &http.RequestConfig{
					Method: "POST",
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)
		})
		Case("client-mosn-server", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedHeader: map[string][]string{
						"server_address": []string{"127.0.0.1:8080"},
					},
					ExpectedBody: []byte("default-http1"),
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)

		})
		Case("server-verify", func() {
			srv := servers[0]
			stats := srv.Stats()
			Verify(stats.ConnectionTotal(), Equal, 1)
			Verify(stats.ConnectionActive(), Equal, 1)
			Verify(stats.ConnectionClosed(), Equal, 0)
			Verify(stats.Requests(), Equal, 2)

		})
	})
}

func genMockHttpServerConfig(serverName string, addr string) string {
	config := lib.Config{
		Protocol: "Http1",
		Config: &http.HttpServerConfig{
			Addr: addr,
			Configs: map[string]*http.ResonseConfig{
				"/": &http.ResonseConfig{
					CommonBuilder: &http.ResponseBuilder{
						StatusCode: 200,
						Header: map[string][]string{
							"server_address": []string{addr},
						},
						Body: "default-http1",
					},
					ErrorBuilder: &http.ResponseBuilder{
						StatusCode: 500, // 500
					},
				},
			},
		},
	}
	s, _ := json.Marshal(config)
	return string(s)
}

const VariableRouterConfigHTTP1 = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "ERROR",
                        "routers": [
                                {
                                        "router_config_name":"router_to_mosn",
                                        "virtual_hosts":[{
                                                "name":"mosn_hosts",
                                                "domains": ["*"],
                                                 "routers": [
                                                        {
                                                                "match":{"prefix":"/"},
                                                                "route":{"cluster_name":"mosn_cluster"}
                                                        }
                                                ]
                                        }]
                                },
                                {
                                        "router_config_name":"router_to_server",
                                        "virtual_hosts":[{
                                                "name":"server_hosts",
                                                "domains": ["*"],
                                                "routers": [
                                                        {
                                                                "match":{
																	"variables":[{
																		"name":"x-mosn-method",
																		"value":"GET",
																		"model":"and"
																	}]
																},
                                                                "route":{"cluster_name":"server_cluster1"}
                                                        },
 														{
                                                                "match":{
																	"variables":[{
																		"name":"x-mosn-method",
																		"value":"POST",
																		"model":"and"
																	}]
																},
                                                                "route":{"cluster_name":"server_cluster2"}
                                                        }
                                                ]
                                        }]
                                }
                        ],
                        "listeners":[
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
                                                                        "router_config_name":"router_to_mosn"
                                                                }
                                                        }
                                                ]
                                        }]
                                },
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
                                "name": "mosn_cluster",
                                "type": "SIMPLE",
                                "lb_type": "LB_RANDOM",
                                "hosts":[
                                        {"address":"127.0.0.1:2046"}
                                ]
                        },
                        {
                                "name": "server_cluster1",
                                "type": "SIMPLE",
                                "lb_type": "LB_RANDOM",
                                "hosts":[
                                        {"address":"127.0.0.1:8080"}
                                ]
                        },
						 {
                                "name": "server_cluster2",
                                "type": "SIMPLE",
                                "lb_type": "LB_RANDOM",
                                "hosts":[
                                        {"address":"127.0.0.1:8081"}
                                ]
                        }
                ]
        }
}`
