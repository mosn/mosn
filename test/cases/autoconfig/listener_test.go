//go:build MOSNTest
// +build MOSNTest

package autoconfig

import (
	"strings"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
)

func TestUpdateListener(t *testing.T) {
	Scenario(t, "test update mosn listener config", func() {
		var m *mosn.MosnOperator
		Setup(func() {
			m = mosn.StartMosn(listenerConfig, "-f", "auto_config=true")
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("add new listener", func() {
			config := `{
				"name": "test_add",
				"address": "127.0.0.1:12345",
				"bind_port": true,
				"filter_chains": [{
					"filters": [{
						"type": "proxy",
						"config": {
							"downstream_protocol": "Http1",
							"upstream_protocol": "Http1",
							"router_config_name":"router_to_server"
						}
					}]
				}]
			}`
			err := m.UpdateConfig(34901, "listener", config)
			Verify(err, Equal, nil)
			content, err := m.GetMosnConfig(34901, "listener=test_add")
			Verify(err, Equal, nil)
			t := strings.Contains(string(content), "127.0.0.1:12345")
			Verify(t, Equal, true)
		})
		Case("update listener", func() {
			config := `{
				"name": "test_add",
				"address": "127.0.0.1:12345",
				"bind_port": true,
				"filter_chains": [{
					"filters": [{
						"type": "proxy",
						"config": {
							"downstream_protocol": "Http1",
							"upstream_protocol": "Http1",
							"router_config_name":"router_to_server"
						}
					}]
				}],
				"stream_filters": [
					{
						"type": "fault"
					}
				]

			}`
			err := m.UpdateConfig(34901, "listener", config)
			Verify(err, Equal, nil)
			// wait auto config dump
			time.Sleep(4 * time.Second)
			mcfg := m.LoadMosnConfig() // read config from files
			var listener *v2.Listener
			for _, ln := range mcfg.Servers[0].Listeners {
				if ln.Name == "test_add" {
					listener = &ln
					break
				}
			}
			Verify(listener, NotNil)
			Verify(len(listener.StreamFilters), Greater, 0)
		})
		TearDown(func() {
			m.Stop()
		})
	})
}

const listenerConfig = `{
	"servers":[
		{
			"default_log_path":"stdout",
			"listeners":[
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
				}
			]
		}
	],
	"admin": {
		"address": {
			"socket_address": {
				"address": "127.0.0.1",
				"port_value": 34901
			}
		}
	}
}`
