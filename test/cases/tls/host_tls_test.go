//go:build MOSNTest
// +build MOSNTest

package tls

import (
	"testing"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/mosn"
	"mosn.io/mosn/test/lib/xprotocol/boltv1"
)

func TestUpdateHostTLSConfig(t *testing.T) {
	Scenario(t, "update host config to tls states", func() {
		var m *mosn.MosnOperator
		m, _ = lib.InitMosn(hostTLSConfig, lib.CreateConfig(MockBoltServerConfig))
		client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
			TargetAddr: "127.0.0.1:2045",
			Verify: &boltv1.VerifyConfig{
				ExpectedStatusCode: 0,
			},
		})
		Case("make tls connection", func() {
			Verify(client.SyncCall(), Equal, true)
			Verify(GetTLSConnpoolMetrics(t, m), Equal, int64(0))
		})
		Case("disable global tls", func() {
			DisableTLS(t, true)
			// request will be failed
			Verify(client.SyncCall(), Equal, false)
			Verify(GetTLSConnpoolMetrics(t, m), Equal, int64(1))
		})
		Case("update router, route to another cluster which tls config is changed", func() {
			config := `{
				"router_config_name":"router_to_mosn",
				"virtual_hosts":[{
					"name":"mosn_hosts",
					"domains": ["*"],
					"routers": [
						{
							"match":{"headers":[{"name":"service","value":".*"}]},
							"route":{"cluster_name":"mosn_cluster_new"}
						}
					]
				}]

			}`
			err := m.UpdateConfig(34901, "router", config)
			Verify(err, Equal, nil)
			Verify(client.SyncCall(), Equal, false)
			Verify(GetTLSConnpoolMetrics(t, m), Equal, int64(2))
		})
		Case("enable global tls", func() {
			DisableTLS(t, false)
			Verify(client.SyncCall(), Equal, false)
			Verify(GetTLSConnpoolMetrics(t, m), Equal, int64(2))
		})
	})
}

const hostTLSConfig = `{
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
								"match":{"headers":[{"name":"service","value":".*"}]},
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
						"tls_context":{
							"status": true,
							"cert_chain": "-----BEGIN CERTIFICATE-----\nMIIDJTCCAg0CAQEwDQYJKoZIhvcNAQELBQAwWzELMAkGA1UEBhMCQ04xCjAIBgNV\nBAgMAWExCjAIBgNVBAcMAWExCjAIBgNVBAoMAWExCjAIBgNVBAsMAWExCjAIBgNV\nBAMMAWExEDAOBgkqhkiG9w0BCQEWAWEwHhcNMTgwNjE0MDMxMzQyWhcNMTkwNjE0\nMDMxMzQyWjBWMQswCQYDVQQGEwJDTjEKMAgGA1UECAwBYTEKMAgGA1UECgwBYTEK\nMAgGA1UECwwBYTERMA8GA1UEAwwIdGVzdC5jb20xEDAOBgkqhkiG9w0BCQEWAWEw\nggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCrPq+Mo0nS3dJU1qGFwlIB\ni9HqRm5RGcfps+0W5LjEhqUKxKUweRrwDaIxpiSqjKeehz9DtLUpXBD29pHuxODU\nVsMH2U1AGWn9l4jMnP6G5iTMPJ3ZTXszeqALe8lm/f807ZA0C7moc+t7/d3+b6d2\nlnwR+yWbIZJUu2qw+HrR0qPpNlBP3EMtlQBOqf4kCl6TfpqrGfc9lW0JjuE6Taq3\ngSIIgzCsoUFe30Yemho/Pp4zA9US97DZjScQLQAGiTsCRDBASxXGzODQOfZL3bCs\n2w//1lqGjmhp+tU1nR4MRN+euyNX42ioEz111iB8y0VzuTIsFBWwRTKK1SF7YSEb\nAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABnRM9JJ21ZaujOTunONyVLHtmxUmrdr\n74OJW8xlXYEMFu57Wi40+4UoeEIUXHviBnONEfcITJITYUdqve2JjQsH2Qw3iBUr\nmsFrWS25t/Krk2FS2cKg8B9azW2+p1mBNm/FneMv2DMWHReGW0cBp3YncWD7OwQL\n9NcYfXfgBgHdhykctEQ97SgLHDKUCU8cPJv14eZ+ehIPiv8cDWw0mMdMeVK9q71Y\nWn2EgP7HzVgdbj17nP9JJjNvets39gD8bU0g2Lw3wuyb/j7CHPBBzqxh+a8pihI5\n3dRRchuVeMQkMuukyR+/A8UrBMA/gCTkXIcP6jKO1SkKq5ZwlMmapPc=\n-----END CERTIFICATE-----\n",
							"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAqz6vjKNJ0t3SVNahhcJSAYvR6kZuURnH6bPtFuS4xIalCsSl\nMHka8A2iMaYkqoynnoc/Q7S1KVwQ9vaR7sTg1FbDB9lNQBlp/ZeIzJz+huYkzDyd\n2U17M3qgC3vJZv3/NO2QNAu5qHPre/3d/m+ndpZ8EfslmyGSVLtqsPh60dKj6TZQ\nT9xDLZUATqn+JApek36aqxn3PZVtCY7hOk2qt4EiCIMwrKFBXt9GHpoaPz6eMwPV\nEvew2Y0nEC0ABok7AkQwQEsVxszg0Dn2S92wrNsP/9Zaho5oafrVNZ0eDETfnrsj\nV+NoqBM9ddYgfMtFc7kyLBQVsEUyitUhe2EhGwIDAQABAoIBAG2Bj5ca0Fmk+hzA\nh9fWdMSCWgE7es4n81wyb/nE15btF1t0dsIxn5VE0qR3P1lEyueoSz+LrpG9Syfy\nc03B3phKxzscrbbAybOeFJ/sASPYxk1IshRE5PT9hJzzUs6mvG1nQWDW4qmjP0Iy\nDKTpV6iRANQqy1iRtlay5r42l6vWwHfRfwAv4ExSS+RgkYcavqOp3e9If2JnFJuo\n7Zds2i7Ux8dURX7lHqKxTt6phgoMmMpvO3lFYVGos7F13OR9NKElzjiefAQbweAt\nt8R+6A1rlIwnfywxET9ZXglfOFK6Q0nqCJhcEcKzT/Xfkd+h9XPACjOObJh3a2+o\nwg9GBFECgYEA2a6JYuFanKzvajFPbSeN1csfI9jPpK2+tB5+BB72dE74B4rjygiG\n0Rb26UjovkYfJJqKuKr4zDL5ziSlJk199Ae2f6T7t7zmyhMlWQtVT12iTQvBINTz\nNerKi5HNVBsCSGj0snbwo8u4QRgTjaIoVqTlOlUQuGqYuZ75l8g35IkCgYEAyWOM\nKagzpGmHWq/0ThN4kkwWOdujxuqrPf4un2WXsir+L90UV7X9wY4mO19pe5Ga2Upu\nXFDsxAZsanf8SbzkTGHvzUobFL7eqsiwaUSGB/cGEtkIyVlAdyW9DhiZFt3i9mEF\nbBsHnEDHPHL4tu+BB8G3WahHjWOnbWZ3NTtP94MCgYEAi3XRmSLtjYER5cPvsevs\nZ7M5oRqvdT7G9divPW6k0MEjEJn/9BjgXqbKy4ylZ/m+zBGinEsVGKXz+wjpMY/m\nCOjEGCUYC5AfgAkiHVkwb6d6asgEFEe6BaoF18MyfBbNsJxlYMzowNeslS+an1vr\nYg9EuMl06+GHNSzPlVl1zZkCgYEAxXx8N2F9eu4NUK4ZafMIGpbIeOZdHbSERp+b\nAq5yasJkT3WB/F04QXVvImv3Gbj4W7r0rEyjUbtm16Vf3sOAMTMdIHhaRCbEXj+9\nVw1eTjM8XoE8b465e92jHk6a2WSvq6IK2i9LcDvJ5QptwZ7uLjgV37L4r7sYtVx0\n69uFGJcCgYEAot7im+Yi7nNsh1dJsDI48liKDjC6rbZoCeF7Tslp8Lt+4J9CA9Ux\nSHyKjg9ujbjkzCWrPU9hkugOidDOmu7tJAxB5cS00qJLRqB5lcPxjOWcBXCdpBkO\n0tdT/xRY/MYLf3wbT95enaPlhfeqBBXKNQDya6nISbfwbMLfNxdZPJ8=\n-----END RSA PRIVATE KEY-----\n"
						},
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
					"insecure_skip": true
				},
				"hosts":[
					{"address":"127.0.0.1:2046"}
				]
			},
			{
				"name": "mosn_cluster_new",
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
