package streamproxy

import (
	"testing"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/http"
	"mosn.io/mosn/test/lib/xprotocol/boltv1"
)

func TestTCPProxy(t *testing.T) {
	Scenario(t, "test tcp proxy http", func() {
		_, _ = lib.InitMosn(TCPProxyConfig, lib.CreateConfig(HttpServerConfig))
		Case("http request", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("http request 2", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
	})
	Scenario(t, "test tcp proxy sofarpc", func() {
		_, _ = lib.InitMosn(TCPProxyConfig, lib.CreateConfig(BoltServerConfig))
		Case("bolt request", func() {
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("bolt request 2", func() {
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})

	})
}

const BoltServerConfig = `{
	"protocol":"bolt",
	"config": {
		"address": "127.0.0.1:8080"
	}
}`

const HttpServerConfig = `{
	"protocol":"Http1",
	"config": {
		"address": "127.0.0.1:8080"
	}
}`

const TCPProxyConfig = `{
	"servers": [
		{
			"default_log_path": "stdout",
			"default_log_level": "ERROR",
			"listeners": [
				{
					 "name": "tcp proxy",
					 "address": "127.0.0.1:2045",
					 "bind_port": true,
					 "filter_chains": [
					 	{
							"filters": [
								{
									"type": "tcp_proxy",
									"config": {
										"routes": [
											{
												"cluster":"serverCluster",
												"source_addrs": [
													{"address":"127.0.0.1","length":24}
												],
												"destination_addrs": [
													{"address":"127.0.0.1","length":24}
												],
												"source_port":"1-65535",
												"destination_port":"1-65535"
											}
										]
									}
								}
							]
						}
					 ]
				},
				{
					"name": "tcp proxy simple",
					"address": "127.0.0.1:2046",
					"bind_port": true,
					"filter_chains": [
						{
							"filters": [
								{
									"type": "tcp_proxy",
									"config": {
										"cluster": "serverCluster"
									}
								}
							]
						}
					]
				}
			]
		}
	],
	"cluster_manager": {
		"clusters": [
			{
				"name":"serverCluster",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
				"hosts": [
					{"address": "127.0.0.1:8080"}
				]
			}
		]
	}
}`
