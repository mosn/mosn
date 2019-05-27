package main

import (
	"fmt"
	"net/http"
	"time"

	"sofastack.io/sofa-mosn/test/lib"
	testlib_http "sofastack.io/sofa-mosn/test/lib/http"
)

/*
Simple http1 proxy used mosn.
The mosn config is client-mosn1-mosn2-server, mosn1-mosn2 used boltv1
The client can request mosn2 directly to test client-mosn-server
*/

/*
Verify:
1. client received data is same as server sended
2. server received request count is same as client sended
3. server create only one connection (with mosn2)
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
												 "match":{
													 "prefix":"/"
												 },
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
									"downstream_protocol": "Http1",
									"upstream_protocol": "Http1",
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
												 "match":{
													 "prefix":"/"
												 },
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

func main() {
	lib.Execute(TestSimpleHttp1)
}

func TestSimpleHttp1() bool {
	fmt.Println("----- Run http1 simple test ")
	// Init
	// start mosn
	mosn := lib.StartMosn(ConfigStr)
	defer mosn.Stop()
	// start a simple boltv1 server
	// the address is same as config (mosn's cluster host address)
	srv := testlib_http.NewMockServer("127.0.0.1:8080", nil)
	go srv.Start()
	// wait server start
	time.Sleep(time.Second)
	// create a simple client config, the address is the mosn listener address

	clientAddrs := []string{
		"127.0.0.1:2045", // client-mosn-mosn-server
		"127.0.0.1:2046", // client-mosn-server
	}
	for _, addr := range clientAddrs {
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
			fmt.Printf("client request %s is failed\n", addr)
			return false
		}
	}
	// Verify the Stats
	connTotal, connActive, connClose := srv.ServerStats.ConnectionStats()
	if !(connTotal == 1 && connActive == 1 && connClose == 0) {
		fmt.Println("server connection is not expected", connTotal, connActive, connClose)
		return false
	}
	if !(srv.ServerStats.RequestStats() == 2 && srv.ServerStats.ResponseStats()[http.StatusOK] == 2) {
		fmt.Println("server request and response is not expected", srv.ServerStats.RequestStats(), srv.ServerStats.ResponseStats())
		return false
	}
	fmt.Println("----- PASS http1 simple test")
	return true
}
