//go:build MOSNTest
// +build MOSNTest

package pluginfilter

import (
	"testing"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/http"
)

func TestSimpleSoPlugin(t *testing.T) {
	Scenario(t, "mosn proxy http request only matched header", func() {
		_, _ = lib.InitMosn(ConfigHttpRoute, lib.CreateConfig(MockHttpServerConfig))
		Case("route success", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Request: &http.RequestConfig{
					Header: map[string][]string{
						"User": []string{"admin"},
					},
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("route failed", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 403,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
	})
}

const MockHttpServerConfig = `{
	"protocol":"Http1",
	"config": {
		"address": "127.0.0.1:8080"
	}
}`

const ConfigHttpRoute = `{
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
                                                                "match":{"prefix":"/"},
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
                                                                        "router_config_name":"router_to_server"
                                                                }
                                                        }
                                                ]
                                        }],
										"stream_filters": [
											{
												"type":"simple_so",
												"go_plugin_config": {
													"so_path":"../../simple_so.so"
												},
												"config": {
													"User":"admin"
												}
											}
										]
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
                        }
                ]
        }
}`
