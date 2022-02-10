// +build MOSNTest

package healthcheck

import (
	"testing"

	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/http"
)

func TestHttpHealthCheck(t *testing.T) {
	Scenario(t, "http health checker with config", func() {
		_, _ = lib.InitMosn(ConfigHttpHealthCluster, lib.CreateConfig(MockHttpServerConfig))
		Case("http health checker fallback to tcp dail", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Request: &http.RequestConfig{
					Header: map[string][]string{
						"service": []string{"*"},
					},
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		//Case("http health checker use rpc port success", func() {
		//	client := lib.CreateClient("Http1", &http.HttpClientConfig{
		//		TargetAddr: "127.0.0.1:2046",
		//		Verify: &http.VerifyConfig{
		//			ExpectedStatusCode: 404,
		//		},
		//	})
		//	Verify(client.SyncCall(), Equal, true)
		//})
		//Case("http health checker success", func() {
		//	client := lib.CreateClient("Http1", &http.HttpClientConfig{
		//		TargetAddr: "127.0.0.1:2046",
		//		Verify: &http.VerifyConfig{
		//			ExpectedStatusCode: 404,
		//		},
		//	})
		//	Verify(client.SyncCall(), Equal, true)
		//})
		//Case("http health checker failed", func() {
		//	client := lib.CreateClient("Http1", &http.HttpClientConfig{
		//		TargetAddr: "127.0.0.1:2046",
		//		Verify: &http.VerifyConfig{
		//			ExpectedStatusCode: 404,
		//		},
		//	})
		//	Verify(client.SyncCall(), Equal, true)
		//})
	})
}

const ConfigHttpHealthCluster = `{
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
                                                                "match":{"headers":[{"name":"service","value":"*"}]},
                                                                "route":{"cluster_name":"local_service"}
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
                                                                        "router_config_name":"local_service"
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
                                "name": "local_service",
                                "type": "SIMPLE",
                                "lb_type": "LB_RANDOM",
                                "hosts":[
                                        {"address":"127.0.0.1:8080"}
                                ],
								"health_check": {
									  "protocol": "Http1",
									  "service_name": "local_service",
									  "timeout": "1s",
									  "interval": "10s",
									  "healthy_threshold": 1,
									  "unhealthy_threshold": 1
									},
                        }
                ]
        }
}`
