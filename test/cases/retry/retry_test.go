// +build MOSNTest

package retry

import (
	"fmt"
	"regexp"
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/extends"
	"mosn.io/mosn/test/lib/http"
)

func TestSimpleRetry(t *testing.T) {
	Scenario(t, "protocol http1 with retry", func() {
		_, servers := lib.InitMosn(CreateRetryConfig(protocol.HTTP1, protocol.HTTP1), lib.CreateConfig(`{
			"protocol":"Http1",
			"config": {
				"address": "127.0.0.1:8080",
				"response_configs": {
					"/": {
						"common_builder": {
							"status_code": 500
						}
					}
				}
			}
		}`), lib.CreateConfig(`{
			"protocol":"Http1",
			"config": {
				"address": "127.0.0.1:8081",
				"response_configs": {
					"/": {
						"common_builder": {
							"status_code": 200
						}
					}
				}
			}
		}`))

		Case("mesh to mesh, one of the upstream server response status code is 500", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			// veirfy, the second server should have 2 requests
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 2)
		})

		Case("mesh to upstream, one of the upstream server response status code is 500", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 4)
		})

		Case("mesh to mesh, one of the upstream server is closed", func() {
			// close the bad server
			badsrv := servers[0]
			badsrv.Stop()
			//
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 6)
		})

		Case("mesh to upstream, one of the upstream server is closed", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 8)
		})
	})

	Scenario(t, "protocol http2 with retry", func() {
		_, servers := lib.InitMosn(CreateRetryConfig(protocol.HTTP2, protocol.HTTP2), lib.CreateConfig(`{
			"protocol":"Http2",
			"config": {
				"address": "127.0.0.1:8080",
				"response_configs": {
					"/": {
						"common_builder": {
							"status_code": 500
						}
					}
				}
			}
		}`), lib.CreateConfig(`{
			"protocol":"Http2",
			"config": {
				"address": "127.0.0.1:8081",
				"response_configs": {
					"/": {
						"common_builder": {
							"status_code": 200
						}
					}
				}
			}
		}`))

		Case("mesh to mesh, one of the upstream server response status code is 500", func() {
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			// veirfy, the second server should have 2 requests
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 2)
		})

		Case("mesh to upstream, one of the upstream server response status code is 500", func() {
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 4)
		})

		Case("mesh to mesh, one of the upstream server is closed", func() {
			// close the bad server
			badsrv := servers[0]
			badsrv.Stop()
			//
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 6)
		})

		Case("mesh to upstream, one of the upstream server is closed", func() {
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 8)
		})

	})
}

func TestRetryWithHttpConvert(t *testing.T) {
	Scenario(t, "protocol http1 trans to http2 with retry", func() {
		_, servers := lib.InitMosn(CreateRetryConfig(protocol.HTTP1, protocol.HTTP2), lib.CreateConfig(`{
			"protocol":"Http1",
			"config": {
				"address": "127.0.0.1:8080",
				"response_configs": {
					"/": {
						"common_builder": {
							"status_code": 500
						}
					}
				}
			}
		}`), lib.CreateConfig(`{
			"protocol":"Http1",
			"config": {
				"address": "127.0.0.1:8081",
				"response_configs": {
					"/": {
						"common_builder": {
							"status_code": 200
						}
					}
				}
			}
		}`))

		Case("mesh to mesh, one of the upstream server response status code is 500", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			// veirfy, the second server should have 2 requests
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 2)
		})

		Case("mesh to mesh, one of the upstream server is closed", func() {
			// close the bad server
			badsrv := servers[0]
			badsrv.Stop()
			//
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 4)
		})

	})

	Scenario(t, "protocol http2 trans to http1 with retry", func() {
		_, servers := lib.InitMosn(CreateRetryConfig(protocol.HTTP2, protocol.HTTP1), lib.CreateConfig(`{
			"protocol":"Http2",
			"config": {
				"address": "127.0.0.1:8080",
				"response_configs": {
					"/": {
						"common_builder": {
							"status_code": 500
						}
					}
				}
			}
		}`), lib.CreateConfig(`{
			"protocol":"Http2",
			"config": {
				"address": "127.0.0.1:8081",
				"response_configs": {
					"/": {
						"common_builder": {
							"status_code": 200
						}
					}
				}
			}
		}`))

		Case("mesh to mesh, one of the upstream server response status code is 500", func() {
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			// veirfy, the second server should have 2 requests
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 2)
		})

		Case("mesh to mesh, one of the upstream server is closed", func() {
			// close the bad server
			badsrv := servers[0]
			badsrv.Stop()
			//
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			// at least run twice
			for i := 0; i < 2; i++ {
				Verify(client.SyncCall(), Equal, true)
			}
			goodsrv := servers[1]
			Verify(goodsrv.Stats().Requests(), Equal, 4)
		})

	})
}

var trimStreamFilters = regexp.MustCompile(`"stream_filters":\[.*\]`)

func CreateRetryConfig(original api.ProtocolName, trans api.ProtocolName) string {
	if original == trans {
		newTmpl := trimStreamFilters.ReplaceAllString(ConfigWithRetryTmpl, `"stream_filters":[]`)
		return fmt.Sprintf(newTmpl, original, original)
	}
	filterin, filtermid := extends.GetTransFilter(original, trans)
	return fmt.Sprintf(ConfigWithRetryTmpl, original, filterin, trans, filtermid)
}

const ConfigWithRetryTmpl = `{
	"servers":[
		{
			"default_log_path":"stdout",
			"default_log_level": "ERROR",
			"routers": [
				{
					"router_config_name":"router_to_upstream",
					"virtual_hosts":[{
						"name":"server_hosts",
						"domains": ["*"],
						"routers": [
							{
								"match":{"prefix":"/"},
								"route":{
									"cluster_name":"upstream",
									"retry_policy": {
										"retry_on": true,
										"retry_timeout": "5s",
										"num_retries": 3
									}
								}
							}
						]
					}]
				},
				{
					"router_config_name":"router_to_mesh",
					"virtual_hosts":[{
						"name":"server_hosts2",
						"domains": ["*"],
						"routers": [
							{
								"match":{"prefix":"/"},
								"route":{
									"cluster_name":"mesh",
									"retry_policy": {
										"retry_on": true,
										"retry_timeout": "5s",
										"num_retries": 3
									}
								}
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
									"downstream_protocol": "%s",
									"upstream_protocol": "Auto",
									"router_config_name":"router_to_mesh"
								}
							}
						]
					}],
					"stream_filters":[{"type": "transcoder","config": {"type": "%s"}}]
				},
				{
					"address":"127.0.0.1:2046",
					"bind_port": true,
					"filter_chains": [{
						"filters": [
							{
								"type": "proxy",
								"config": {
									"downstream_protocol": "%s",
									"upstream_protocol": "Auto",
									"router_config_name":"router_to_upstream"
								}
							}
						]
					}],
					"stream_filters":[{"type": "transcoder","config": {"type": "%s"}}]
				}
			]
		}
	],
	"cluster_manager":{
		"clusters":[
			{
				"name": "mesh",
				"type": "SIMPLE",
				"lb_type": "LB_ROUNDROBIN",
				"hosts":[
					{"address":"127.0.0.1:2046"}
				]
			},
			{
				"name": "upstream",
				"type": "SIMPLE",
				"lb_type": "LB_ROUNDROBIN",
				"hosts":[
					{"address":"127.0.0.1:8080"},
					{"address":"127.0.0.1:8081"}
				]
			}
		]
	}
}`
