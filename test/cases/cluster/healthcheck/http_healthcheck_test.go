//go:build MOSNTest
// +build MOSNTest

package healthcheck

import (
	"testing"
	"time"

	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/http"

	. "mosn.io/mosn/test/framework"
)

func TestHttpHealthCheck(t *testing.T) {
	Scenario(t, "http health checker without config", func() {
		_, _ = lib.InitMosn(ConfigHttpHealthCluster, lib.CreateConfig(MockHttpCheckServerConfig))
		Case("http health checker fallback to tcp dail", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Request: &http.RequestConfig{
					Header: map[string][]string{
						"x-test-route": []string{"paas"},
					},
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			time.Sleep(time.Second * 3)
			Verify(client.SyncCall(), Equal, true)
		})
	})
}

func TestHttpHealthCheckWithConfigFailed(t *testing.T) {
	Scenario(t, "http health checker with config error", func() {
		_, _ = lib.InitMosn(ConfigHttpHealthCluster2, lib.CreateConfig(MockHttpCheckServerConfig))
		Case("http health checker fallback to tcp dail", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Request: &http.RequestConfig{
					Header: map[string][]string{
						"x-test-route": []string{"paas"},
					},
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 502,
				},
			})
			time.Sleep(time.Second * 3)
			Verify(client.SyncCall(), Equal, true)
		})
	})
}

func TestHttpHealthCheckWithConfigSuccess(t *testing.T) {
	Scenario(t, "http health checker with config right", func() {
		_, _ = lib.InitMosn(ConfigHttpHealthCluster3, lib.CreateConfig(MockHttpCheckServerConfig))
		Case("http health checker fallback to tcp dail", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Request: &http.RequestConfig{
					Header: map[string][]string{
						"x-test-route": []string{"paas"},
					},
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			time.Sleep(time.Second * 3)
			Verify(client.SyncCall(), Equal, true)
		})
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
                                ],
								"health_check": {
									  "protocol": "Http1",
									  "service_name": "local_service",
									  "timeout": "1s",
									  "interval": "1s",
									  "healthy_threshold": 1,
									  "unhealthy_threshold": 1
									}
                        }
                ]
        }
}`

const ConfigHttpHealthCluster2 = `{
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
                                ],
								"health_check": {
									  "protocol": "Http1",
									  "service_name": "local_service",
									  "timeout": "1s",
									  "interval": "1s",
									  "healthy_threshold": 1,
									  "unhealthy_threshold": 1,
									  "check_config": {
											"http_check_config":{
												"port":12133
											}
										}
									}
                        }
                ]
        }
}`

const ConfigHttpHealthCluster3 = `{
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
                                ],
								"health_check": {
									  "protocol": "Http1",
									  "service_name": "local_service",
									  "timeout": "1s",
									  "interval": "1s",
									  "healthy_threshold": 1,
									  "unhealthy_threshold": 1,
									  "check_config": {
											"http_check_config":{
												"port":8080,
												"timeout": "30s",
												"path": ""
											}
										}
									}
                        }
                ]
        }
}`

const MockHttpCheckServerConfig = `{
	"protocol":"Http1",
	"config": {
		"address": "127.0.0.1:8080"
	}
}`
