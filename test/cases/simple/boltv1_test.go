//go:build MOSNTest
// +build MOSNTest

package simple

import (
	"testing"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/xprotocol/boltv1"
)

func TestSimpleBoltv1(t *testing.T) {
	Scenario(t, "simple boltv1 proxy used mosn.", func() {
		// servers is invalid in `Case`
		_, servers := lib.InitMosn(ConfigSimpleBoltv1, lib.CreateConfig(MockBoltServerConfig))
		Case("client-mosn-mosn-server", func() {
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)
		})
		Case("client-mosn-server", func() {
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)
		})
		Case("server-verify", func() {
			srv := servers[0]
			stats := srv.Stats()
			Verify(stats.ConnectionTotal(), Equal, 1)
			Verify(stats.ConnectionActive(), Equal, 1)
			Verify(stats.ConnectionClosed(), Equal, 0)
			Verify(stats.Requests(), Equal, 2)
		})
	})
}

const MockBoltServerConfig = `{
	"protocol":"bolt",
	"config": {
		"address": "127.0.0.1:8080"
	}
}`

const ConfigSimpleBoltv1 = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "ERROR",
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
        "cluster_manager":{
                "clusters":[
                        {
                                "name": "mosn_cluster",
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
        }
}`
