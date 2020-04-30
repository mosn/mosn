// +build MOSNTest

package simple

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/sofarpc"
)

func runSimpleClient(addr string) error {
	cfg := sofarpc.CreateSimpleConfig(addr)
	// the simple server's response is:
	// Header mosn-test-default: boltv1
	// Content: default-boltv1
	// we will set the client's verify
	VefiyCfg := &sofarpc.VerifyConfig{
		ExpectedStatus: bolt.ResponseStatusSuccess,
		ExpectedHeader: map[string]string{
			"mosn-test-default": "boltv1",
		},
		ExpectedBody: []byte("default-boltv1"),
	}
	cfg.Verify = VefiyCfg.Verify
	// create only one connection
	clt := sofarpc.NewClient(cfg, 1)
	// send a request, and verify the result
	if !clt.SyncCall() {
		return errors.New(fmt.Sprintf("client request %s is failed", addr))
	}
	return nil
}

func TestSimpleBoltv1(t *testing.T) {
	lib.Scenario(t, "Simple boltv1 proxy used mosn.", func() {
		var mosn *lib.MosnOperator
		var srv *sofarpc.MockServer
		lib.Setup(func() error {
			// start mosn
			mosn = lib.StartMosn(ConfigSimpleBoltv1)
			// start a simple boltv1 server
			// the address is same as config (mosn's cluster host address)
			srv = sofarpc.NewMockServer("127.0.0.1:8080", nil)
			go srv.Start()
			time.Sleep(time.Second) // wait server start
			return nil
		})
		lib.TearDown(func() {
			srv.Close()
			mosn.Stop()
		})
		lib.Execute("client-mosn-mosn-server", func() error {
			addr := "127.0.0.1:2045"
			return runSimpleClient(addr)
		})
		lib.Execute("client-mosn-server", func() error {
			addr := "127.0.0.1:2046"
			return runSimpleClient(addr)
		})
		lib.Verify(func() error {
			connTotal, connActive, connClose := srv.ServerStats.ConnectionStats()
			if !(connTotal == 1 && connActive == 1 && connClose == 0) {
				msg := fmt.Sprintf("server connection is not expected, %d, %d, %d", connTotal, connActive, connClose)
				return errors.New(msg)
			}
			if !(srv.ServerStats.RequestStats() == 2 && srv.ServerStats.ResponseStats()[int16(bolt.ResponseStatusSuccess)] == 2) {
				msg := fmt.Sprintf("server request and response is not expected, %d, %d", srv.ServerStats.RequestStats(), srv.ServerStats.ResponseStats())
				return errors.New(msg)
			}
			return nil
		})
	})
}

const ConfigSimpleBoltv1 = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "DEBUG",
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
