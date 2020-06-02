// +build MOSNTest

package directresp

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"mosn.io/mosn/test/lib"
	testlib_http "mosn.io/mosn/test/lib/http"
)

func TestHttp1DirectResponse(t *testing.T) {
	lib.Scenario(t, "Direct response in http1", func() {
		var mosn *lib.MosnOperator
		lib.Setup(func() error {
			mosn = lib.StartMosn(ConfigHttp1)
			time.Sleep(time.Second)
			return nil
		})
		lib.TearDown(func() {
			mosn.Stop()
		})
		lib.Execute("get response from mosn", func() error {
			cltVerify := &testlib_http.VerifyConfig{
				ExpectedStatus: http.StatusOK,
				ExpectedBody:   []byte("test body"),
			}
			cfg := testlib_http.CreateSimpleConfig("127.0.0.1:2045")
			cfg.Verify = cltVerify.Verify
			clt := testlib_http.NewClient(cfg, 1)
			if !clt.SyncCall() {
				return errors.New("client receive response unexpected")
			}
			return nil
		})
	})
}

const ConfigHttp1 = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "FATAL",
                        "routers":[
                                {
                                        "router_config_name":"router_direct",
                                        "virtual_hosts":[{
                                                "name":"mosn_hosts",
                                                "domains": ["*"],
                                                "routers": [
                                                        {
                                                                "match":{"prefix": "/"},
                                                                 "direct_response": {
                                                                         "status": 200,
                                                                         "body": "test body"
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
                                        "log_path": "stdout",
                                        "log_level": "FATAL",
                                        "filter_chains": [{
                                                "filters": [
                                                        {
                                                                "type": "proxy",
                                                                "config": {
                                                                        "downstream_protocol": "Http1",
                                                                        "upstream_protocol": "Http1",
                                                                        "router_config_name":"router_direct"
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
                                "name": "empty_cluster",
                                "type": "SIMPLE"
                        }
                ]
        }
}`
