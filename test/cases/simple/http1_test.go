// +build MOSNTest

package simple

import (
	"testing"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/http"
)

func TestSimpleHTTP1(t *testing.T) {
	Scenario(t, "simple http1 proxy used mosn", func() {
		// servers is invalid in `Case`
		_, servers := lib.InitMosn(ConfigSimpleHTTP1, lib.CreateConfig(MockHttpServerConfig))
		Case("client-mosn-mosn-server", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedHeader: map[string][]string{
						"mosn-test-default": []string{"http1"},
					},
					ExpectedBody: []byte("default-http1"),
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
						"mosn-test-default": []string{"http1"},
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

const MockHttpServerConfig = `{
	"protocol":"Http1",
	"config": {
		"address": "127.0.0.1:8080"
	}
}`

const ConfigSimpleHTTP1 = `{
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
                                                                "match":{"prefix":"/"},
                                                                "route":{"cluster_name":"server_cluster"}
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
