// +build MOSNTest

package simple

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"mosn.io/mosn/test/lib"
	testlib_http "mosn.io/mosn/test/lib/http"
)

func runSimpleHttpClient(addr string) error {
	cfg := testlib_http.CreateSimpleConfig(addr)
	VefiyCfg := &testlib_http.VerifyConfig{
		ExpectedStatus: http.StatusOK,
		ExpectedHeader: map[string]string{
			"mosn-test-default": "http1",
		},
		ExpectedBody: []byte("default-http1"),
	}
	cfg.Verify = VefiyCfg.Verify
	// create only one connection
	clt := testlib_http.NewClient(cfg, 1)
	// send a request, and verify the result
	if !clt.SyncCall() {
		return errors.New(fmt.Sprintf("client request %s is failed", addr))
	}
	return nil
}

func TestSimpleHTTP1(t *testing.T) {
	lib.Scenario(t, "Simple http1 proxy used mosn", func() {
		var mosn *lib.MosnOperator
		var srv *testlib_http.MockServer
		lib.Setup(func() error {
			mosn = lib.StartMosn(ConfigSimpleHTTP1)
			srv = testlib_http.NewMockServer("127.0.0.1:8080", nil)
			go srv.Start()
			time.Sleep(time.Second)
			return nil
		})
		lib.TearDown(func() {
			srv.Close()
			mosn.Stop()
		})
		lib.Execute("client-mosn-mosn-server", func() error {
			return runSimpleHttpClient("127.0.0.1:2045")
		})
		lib.Execute("client-mosn-server", func() error {
			return runSimpleHttpClient("127.0.0.1:2046")
		})
		lib.Verify(func() error {
			connTotal, connActive, connClose := srv.ServerStats.ConnectionStats()
			if !(connTotal == 1 && connActive == 1 && connClose == 0) {
				msg := fmt.Sprintf("server connection is not expected %d, %d, %d", connTotal, connActive, connClose)
				return errors.New(msg)
			}
			if !(srv.ServerStats.RequestStats() == 2 && srv.ServerStats.ResponseStats()[http.StatusOK] == 2) {
				msg := fmt.Sprintf("server request and response is not expected %d, %d", srv.ServerStats.RequestStats(), srv.ServerStats.ResponseStats())
				return errors.New(msg)
			}
			return nil
		})
	})
}

const ConfigSimpleHTTP1 = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "FATAL",
                        "routers": [
                                {
                                        "router_config_name":"router_to_mosn",
                                        "virtual_hosts":[{
                                                "name":"mosn_hosts",
                                                "domains": ["*"],
                                                "routers": [
                                                        {
                                                                "match":{"prefix":"/"},
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
                                                                "match":{"prefix":"/"},
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
                                        "log_path": "stdout",
                                        "log_level": "FATAL",
                                        "filter_chains": [{
                                                "filters": [
                                                        {
                                                                "type": "proxy",
                                                                "config": {
                                                                        "downstream_protocol": "Http1",
                                                                        "upstream_protocol": "Http1",
                                                                        "router_config_name":"router_to_mosn"
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
                                                                        "downstream_protocol": "Http1",
                                                                        "upstream_protocol": "Http1",
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
