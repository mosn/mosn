//go:build MOSNTest
// +build MOSNTest

package tls

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

func TestTLSExtendMode(t *testing.T) {
	Scenario(t, "protocol http1 with tls extend", func() {
		lib.InitMosn(CreateTlsConfig(protocol.HTTP1, protocol.HTTP1), lib.CreateConfig(`{
			"protocol":"Http1",
			"config": {
				"address": "127.0.0.1:8080"
			}
		}`))
		Case("http1 with tls extend", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
	})

	Scenario(t, "protocol http2 with tls extend", func() {
		lib.InitMosn(CreateTlsConfig(protocol.HTTP2, protocol.HTTP2), lib.CreateConfig(`{
			"protocol":"Http2",
			"config": {
				"address": "127.0.0.1:8080"
			}
		}`))
		Case("http2 with tls extend", func() {
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
	})

	Scenario(t, "protocol http1 to http2 with tls extend", func() {
		lib.InitMosn(CreateTlsConfig(protocol.HTTP1, protocol.HTTP2), lib.CreateConfig(`{
			"protocol":"Http1",
			"config": {
				"address": "127.0.0.1:8080"
			}
		}`))
		Case("http1 to http2 with tls extend", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
	})

	Scenario(t, "protocol http2 to http1 with tls extend", func() {
		lib.InitMosn(CreateTlsConfig(protocol.HTTP2, protocol.HTTP1), lib.CreateConfig(`{
			"protocol":"Http2",
			"config": {
				"address": "127.0.0.1:8080"
			}
		}`))
		Case("http2 with tls extend", func() {
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
	})

}

var trimStreamFilters = regexp.MustCompile(`"stream_filters":\[.*\]`)

func CreateTlsConfig(original api.ProtocolName, trans api.ProtocolName) string {
	if original == trans {
		newTmpl := trimStreamFilters.ReplaceAllString(ConfigTlsExtendConfigTmpl, `"stream_filters":[]`)
		return fmt.Sprintf(newTmpl, original, original)
	}
	filterin, filtermid := extends.GetTransFilter(original, trans)
	return fmt.Sprintf(ConfigTlsExtendConfigTmpl, original, filterin, trans, filtermid)
}

const ConfigTlsExtendConfigTmpl = `{
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
								"route":{"cluster_name":"upstream"}
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
								"route":{"cluster_name":"mesh"}
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
						"tls_context":{
							"status": true,
							"type": "mosn-integrate-test-tls"
						},
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
				"tls_context": {
					"status": true,
					"type": "mosn-integrate-test-tls"
				},
				"hosts":[
					{"address":"127.0.0.1:2046"}
				]
			},
			{
				"name": "upstream",
				"type": "SIMPLE",
				"lb_type": "LB_ROUNDROBIN",
				"hosts":[
					{"address":"127.0.0.1:8080"}
				]
			}
		]
	}
}`
