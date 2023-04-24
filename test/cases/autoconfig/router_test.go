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

func TestUpdateRouter(t *testing.T) {
	Scenario(t, "test update mosn router config", func() {
		var m *mosn.MosnOperator
		Setup(func() {
			m = mosn.StartMosn(routerConfig, "-f", "auto_config=true")
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("add new router", func() {
			config := `{
				"router_config_name": "test_router",
				"virtual_hosts": [
					{
						"name":"serverHost",
						"domains": ["*"],
						"routers": [
							{
								"match":{"prefix":"/"},
								"route":{"cluster_name":"test"}
							}
						]
					}
				]
			}`
			err := m.UpdateConfig(34901, "router", config)
			Verify(err, Equal, nil)
			content, err := m.GetMosnConfig(34901, "router=test_router")
			Verify(err, Equal, nil)
			t := strings.Contains(string(content), "prefix")
			Verify(t, Equal, true)
		})
		Case("update router", func() {
			config := `{
				"router_config_name": "test_router",
				"virtual_hosts": [
					{
						"name":"serverHost",
						"domains": ["*"],
						"routers": [
							{
								"match":{"path":"/test"},
								"route":{"cluster_name":"test2"}
							},
							{
								"match":{"prefix":"/"},
								"route":{"cluster_name":"test"}
							}
							
						]
					}
				]
			}`
			err := m.UpdateConfig(34901, "router", config)
			Verify(err, Equal, nil)
			// wait auto config dump
			time.Sleep(4 * time.Second)
			mcfg := m.LoadMosnConfig() // read config from files
			var route *v2.RouterConfiguration
			for _, r := range mcfg.Servers[0].Routers {
				if r.RouterConfigName == "test_router" {
					route = r
				}
			}
			Verify(route, NotNil)
			Verify(len(route.VirtualHosts[0].Routers), Equal, 2)
		})
		Case("add route rule", func() {
			config := `{
				"router_config_name": "test_router",
				"domain": "",
				"route": {
					"match":{"path":"/new"},
					"route":{"cluster_name":"testnew"}
				}
			}`
			err := m.UpdateRoute(34901, "add", config)
			Verify(err, Equal, nil)
			// wait auto config dump
			time.Sleep(4 * time.Second)
			mcfg := m.LoadMosnConfig() // read config from files
			var route *v2.RouterConfiguration
			for _, r := range mcfg.Servers[0].Routers {
				if r.RouterConfigName == "test_router" {
					route = r
				}
			}
			Verify(route, NotNil)
			Verify(len(route.VirtualHosts[0].Routers), Equal, 3)
		})
		Case("remove route rule", func() {
			config := `{
				"router_config_name": "test_router",
				"domain": ""
			}`
			err := m.UpdateRoute(34901, "remove", config)
			Verify(err, Equal, nil)
			// wait auto config dump
			time.Sleep(4 * time.Second)
			mcfg := m.LoadMosnConfig() // read config from files
			var route *v2.RouterConfiguration
			for _, r := range mcfg.Servers[0].Routers {
				if r.RouterConfigName == "test_router" {
					route = r
				}
			}
			Verify(route, NotNil)
			Verify(len(route.VirtualHosts[0].Routers), Equal, 0)
		})
		TearDown(func() {
			m.Stop()
		})
	})
}

const routerConfig = `{
	"servers":[
		{
			"default_log_path":"stdout"
		}
	],
	"cluster_manager": {},
	"admin": {
		"address": {
			"socket_address": {
				"address": "127.0.0.1",
				"port_value": 34901
			}
		}
	}
}`
