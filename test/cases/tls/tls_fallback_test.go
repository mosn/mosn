//go:build MOSNTest
// +build MOSNTest

package tls

import (
	"testing"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/xprotocol/boltv1"
)

func TestServerNotSupportTLS(t *testing.T) {
	Scenario(t, "server side mosn not support tls, but client request with tls", func() {
		lib.InitMosn(mistakeTLSConfig, lib.CreateConfig(MockBoltServerConfig))
		Case("client request tls, trigger fallback", func() {
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Request: &boltv1.RequestConfig{
					Header: map[string]string{
						"service": "fallback",
					},
				},
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("client rquest tls, no fallback", func() {
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Request: &boltv1.RequestConfig{
					Header: map[string]string{
						"service": "test",
					},
				},
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 6, // expected got a ResponseStatusNoProcessor
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
	})
}

const mistakeTLSConfig = `{
	"servers":[
		{
			"default_log_path":"stdout",
			"default_log_level": "INFO",
			"routers":[
				{
					"router_config_name":"router_to_mosn",
					"virtual_hosts":[{
						"name":"mosn_hosts",
						"domains": ["*"],
						"routers": [
							{
								"match":{"headers":[{"name":"service","value":"fallback"}]},
								"route":{"cluster_name":"mosn_cluster"}
							},
							{
								 "match":{"headers":[{"name":"service","value":"test"}]},
								 "route":{"cluster_name":"mosn_cluster2"}
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
								"match":{"headers":[{"name":"service","value":".*"}]},
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
									"downstream_protocol": "X",
									"upstream_protocol": "X",
									"extend_config": {
										 "sub_protocol": "bolt"
									},
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
									 "downstream_protocol": "X",
									 "upstream_protocol": "X",
									 "extend_config": {
										 "sub_protocol": "bolt"
									 },
									 "router_config_name":"router_to_server"
								 }
							}
						]
					}]
				},
				{
					"address":"127.0.0.1:2047",
					"bind_port": true,
					"filter_chains": [{
						"filters": [
							{
								"type": "proxy",
								"config": {
									"downstream_protocol": "X",
									"upstream_protocol": "X",
									"extend_config": {
										 "sub_protocol": "bolt"
									},
									"router_config_name":"router_to_server"
								}
							}
						]
					}]
				}
			]
		}
	],
	"cluster_manager": {
		"clusters":[
			{
				"name": "mosn_cluster",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
				"tls_context": {
					"status": true,
					"insecure_skip": true,
					"fall_back": true
				},
				"hosts":[
					{"address":"127.0.0.1:2046"}
				]
			},
			{
				"name": "mosn_cluster2",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
				"tls_context": {
					"status": true,
					"insecure_skip": true
				},
				"hosts":[
					{"address":"127.0.0.1:2047"}
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
	},
	"admin": {
		"address": {
			"socket_address": {
				"address": "127.0.0.1",
				"port_value": 34901
			}
		}
	}
}`
