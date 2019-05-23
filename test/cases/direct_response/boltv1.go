package main

import (
	"fmt"
	"time"

	"sofastack.io/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"sofastack.io/sofa-mosn/test/lib"
	testlib_sofarpc "sofastack.io/sofa-mosn/test/lib/sofarpc"
)

/*
DirectResponse Case.
If a request matched route with direct response, mosn will send a response directly.
Use HTTP Status Code as standard
TODO: fix direct response cannot send body in rpc protocol
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
									"downstream_protocol": "SofaRpc",
									"upstream_protocol": "SofaRpc",
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
												 "match":{"headers":[{"name":"service","value":".*"}]},
												 "direct_response": {
													 "status": 200
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
	fmt.Println("----- Run boltv1 direct response test ")

	mosn := lib.StartMosn(ConfigStr)
	defer mosn.Stop()

	// wait mosn start
	time.Sleep(time.Second)

	// direct response, no server needed
	// client config
	// TODO: mosn support send header/body
	cltVerify := &testlib_sofarpc.VerifyConfig{
		ExpectedStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
	}
	cfg := testlib_sofarpc.CreateSimpleConfig("127.0.0.1:2045")
	cfg.Verify = cltVerify.Verify

	clt := testlib_sofarpc.NewClient(cfg, 1)
	if !clt.SyncCall() {
		fmt.Println("client receive response unexpected")
		return false
	}
	fmt.Println("----- PASS boltv1 direct response test ")

	return true

}

func main() {
	lib.Execute(TestDirectResponse)
}
