package main

import (
	"fmt"
	"net/http"
	"time"

	"sofastack.io/sofa-mosn/test/lib"
	testlib_http "sofastack.io/sofa-mosn/test/lib/http"
)

/*
DirectResponse Case.
If a request matched route with direct response, mosn will send a response directly.
*/

const ConfigStr = `{
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
                                                                        "downstream_protocol": "Http1",
                                                                        "upstream_protocol": "Http1",
                                                                        "router_config_name":"router_direct"
                                                                }
                                                        },
                                                        {
                                                                "type": "connection_manager",
                                                                "config": {
                                                                        "router_config_name":"router_direct",
                                                                        "virtual_hosts":[{
                                                                                "name":"mosn_hosts",
                                                                                "domains": ["*"],
                                                                                "routers": [
                                                                                        {
                                                                                                 "match":{
													 "prefix": "/"
												 },
                                                                                                 "direct_response": {
                                                                                                         "status": 200,
													 "body": "test body"
                                                                                                 }
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
                                "name": "empty_cluster",
                                "type": "SIMPLE"
                        }
                ]
        }
}`

func TestDirectResponse() bool {
	fmt.Println("----- Run http1 direct response test ")

	mosn := lib.StartMosn(ConfigStr)
	defer mosn.Stop()
	// wait mosn start
	time.Sleep(time.Second)
	// direct response, no server needed
	cltVerify := &testlib_http.VerifyConfig{
		ExpectedStatus: http.StatusOK,
		ExpectedBody:   []byte("test body"),
	}
	cfg := testlib_http.CreateSimpleConfig("127.0.0.1:2045")
	cfg.Verify = cltVerify.Verify
	clt := testlib_http.NewClient(cfg, 1)
	if !clt.SyncCall() {
		fmt.Println("client receive response unexpected")
		return false
	}
	fmt.Println("----- PASS http1 direct response test ")

	return true
}

func main() {
	lib.Execute(TestDirectResponse)
}
