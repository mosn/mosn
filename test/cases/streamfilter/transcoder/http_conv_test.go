// +build MOSNTest

package transcoder

import (
	"fmt"
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/extends"
	"mosn.io/mosn/test/lib/http"
)

func TestDefaultHTTPConvert(t *testing.T) {
	Scenario(t, "mesh convert protocol: http", func() {
		lib.InitMosn(CreateProtocolConvertConfig(protocol.HTTP1, protocol.HTTP2), lib.CreateConfig(`{
				"protocol":"Http1",
				"config": {
					"address": "127.0.0.1:8080"
				}
			}`), lib.CreateConfig(`{
				"protocol":"Http2",
				"config": {
					"address":"127.0.0.1:8081",
					"response_configs": {
						"/": {
							"common_builder": {
								"status_code": 200,
								"header": {
									"x-mosn-proto": ["http2"]
								},
								"body": "default-http2"
							}
						},
						"/upstream": {
							"common_builder": {
								"status_code": 200,
								"header": {
									"x-mosn-proto": ["http2"]
								},
								"body": "upstream-http2"
							}
						}
					}
				}
			}`))
		Case("mesh to mesh, upstream server is http1", func() {
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
		})
		Case("mesh to server, upstream server is http2", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Request: &http.RequestConfig{
					Method: "GET",
					Path:   "/upstream",
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedHeader: map[string][]string{
						"x-mosn-proto": []string{
							"http2",
						},
					},
					ExpectedBody: []byte("upstream-http2"),
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("mesh direct response", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Request: &http.RequestConfig{
					Method: "GET",
					Path:   "/direct_response",
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedBody:       []byte("mosn direct response"),
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
	})
}

func TestDefaultHTTP2Convert(t *testing.T) {
	Scenario(t, "mesh convert protocol: http2", func() {
		lib.InitMosn(CreateProtocolConvertConfig(protocol.HTTP2, protocol.HTTP1), lib.CreateConfig(`{
				"protocol":"Http1",
				"config": {
					"address": "127.0.0.1:8081",
					"response_configs": {
						"/": {
							 "common_builder": {
								 "status_code": 200,
								 "header": {
									 "x-mosn-proto": ["http1"]
								 },
								 "body": "default-http1"
							 }
						},
						"/upstream": {
							"common_builder": {
								"status_code": 200,
								"header": {
									"x-mosn-proto": ["http1"]
								},
								"body": "upstream-http1"
							}
						}
					}
				}
			}`), lib.CreateConfig(`{
				"protocol":"Http2",
				"config": {
					"address":"127.0.0.1:8080",
					"response_configs": {
						"/": {
							"common_builder": {
								"status_code": 200,
								"header": {
									"x-mosn-proto": ["http2"]
								},
								"body": "default-http2"
							}
						}
					}
				}
			}`))
		Case("mesh to mesh, upstream server is http2", func() {
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedHeader: map[string][]string{
						"x-mosn-proto": []string{"http2"},
					},
					ExpectedBody: []byte("default-http2"),
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("mesh to server, upstream server is http1", func() {
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Request: &http.RequestConfig{
					Method: "GET",
					Path:   "/upstream",
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedHeader: map[string][]string{
						"x-mosn-proto": []string{"http1"},
					},
					ExpectedBody: []byte("upstream-http1"),
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("mesh direct response", func() {
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Request: &http.RequestConfig{
					Method: "GET",
					Path:   "/direct_response",
				},
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedBody:       []byte("mosn direct response"),
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})

	})

}

func CreateProtocolConvertConfig(original, trans api.ProtocolName) string {
	filterin, filtermid := extends.GetTransFilter(original, trans)
	return fmt.Sprintf(ConfigProtocolConvertTmpl, original, filterin, trans, filtermid)
}

const ConfigProtocolConvertTmpl = `{
	"servers":[
		{
			"default_log_path":"stdout",
			"default_log_level": "ERROR",
			"routers": [
				{
					"router_config_name":"client_in",
					"virtual_hosts":[{
						"name":"server_hosts",
						"domains": ["*"],
						"routers": [
							{
								"match":{"prefix":"/direct_response"},
								"direct_response": {
									"status": 200,
									"body": "mosn direct response"
								}
							},
							{
								"match":{"prefix":"/upstream"},
								"route":{"cluster_name":"trans_server"}
							},
							{
								"match":{"prefix":"/"},
								"route":{"cluster_name":"to_mesh"}
							}
						]
					}]
				},
				{
					"router_config_name":"router_to_server",
					"virtual_hosts":[{
						"name":"server_hosts_2",
						"domains": ["*"],
						"routers": [
							{
								"match":{"prefix":"/"},
								"route":{"cluster_name":"original_server"}
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
									"router_config_name":"client_in"
								}
							}
						]
					}],
					"stream_filters": [
						{
							 "type": "transcoder",
							 "config": {
								 "type": "%s"
							 }
						}
					]
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
									"router_config_name":"router_to_server"
								}
							}
						]
					}],
					"stream_filters": [
						{
							"type": "transcoder",
							"config": {
								"type": "%s"
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
				"name": "original_server",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
				"hosts":[
					{"address":"127.0.0.1:8080"}
				]
			},
			{
				"name": "to_mesh",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
				"hosts":[
					{"address":"127.0.0.1:2046"}
				]
			},
			{
				"name": "trans_server",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
				"hosts":[
					{"address":"127.0.0.1:8081"}
				]
			}
		]
	}
}`
