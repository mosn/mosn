package simple

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol/rpc/sofarpc"
	"mosn.io/mosn/test/lib"
	testlib_sofarpc "mosn.io/mosn/test/lib/sofarpc"
)

func TestBoltv1Convert(t *testing.T) {
	convertList := []string{
		"Http1",
		"Http2",
	}
	for _, proto := range convertList {
		msg := fmt.Sprintf("boltv1 convert to %s", proto)
		lib.Scenario(t, msg, func() {
			var mosn *lib.MosnOperator
			var srv *testlib_sofarpc.MockServer
			lib.Setup(func() error {
				configStr := fmt.Sprintf(ConfigBoltv1ConvertTmpl, proto, proto)
				mosn = lib.StartMosn(configStr)
				srv = testlib_sofarpc.NewMockServer("127.0.0.1:8080", nil)
				go srv.Start()
				time.Sleep(time.Second)
				return nil
			})
			lib.TearDown(func() {
				srv.Close()
				mosn.Stop()
			})
			lib.Execute("request mosn", func() error {
				return runSimpleClient("127.0.0.1:2045")
			})
			lib.Verify(func() error {
				connTotal, connActive, connClose := srv.ServerStats.ConnectionStats()
				if !(connTotal == 1 && connActive == 1 && connClose == 0) {
					errmsg := fmt.Sprintf("server connection is not expected %d, %d, %d", connTotal, connActive, connClose)
					return errors.New(errmsg)
				}
				if !(srv.ServerStats.RequestStats() == 1 && srv.ServerStats.ResponseStats()[sofarpc.RESPONSE_STATUS_SUCCESS] == 1) {
					errmsg := fmt.Sprintf("server request and response is not expected %d, %d", srv.ServerStats.RequestStats(), srv.ServerStats.ResponseStats())
					return errors.New(errmsg)
				}
				return nil
			})
		})
	}
}

const ConfigBoltv1ConvertTmpl = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "FATAL",
                        "listeners":[
                                {
                                        "address":"127.0.0.1:2045",
                                        "bind_port": true,
                                        "log_path": "stdout",
                                        "log_level": "FATAL",
                                        "filter_chains": [{
                                                "filters": [
                                                        {
                                                                "type": "proxy",
                                                                "config": {
                                                                        "downstream_protocol": "SofaRpc",
                                                                        "upstream_protocol": "%s",
                                                                        "router_config_name":"router_to_mosn"
                                                                }
                                                        },
                                                        {
                                                                "type": "connection_manager",
                                                                "config": {
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
                                                                }
                                                        }
                                                ]
                                        }]
                                },
                                {
                                        "address":"127.0.0.1:2046",
                                        "bind_port": true,
                                        "log_path": "stdout",
                                        "log_LEVEL": "FATAL",
                                        "filter_chains": [{
                                                "filters": [
                                                        {
                                                                "type": "proxy",
                                                                "config": {
                                                                        "downstream_protocol": "%s",
                                                                        "upstream_protocol": "SofaRpc",
                                                                        "router_config_name":"router_to_server"
                                                                }
                                                      },
                                                        {
                                                                "type": "connection_manager",
                                                                "config": {
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
