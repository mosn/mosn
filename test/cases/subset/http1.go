package main

import (
	"fmt"
	"net/http"
	"time"

	"sofastack.io/sofa-mosn/test/lib"
	testlib_http "sofastack.io/sofa-mosn/test/lib/http"
)

/*
Cluster have two subsets, each subset have one host(upstream server)
upstream server in different subset expected receive different header and do different response.
different request will route to different upstream server, and want to receivee different response.(same as server send)
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
													 "prefix": "/",
													 "headers":[{"name":"service","value":"1.0"}]
												 },
                                                                                                 "route":{
													"cluster_name":"server_cluster",
													"metadata_match": {
														"filter_metadata": {
															"mosn.lb": {
																"version":"1.0"
															}
														}
													}
												}
                                                                                        },
											{
												"match":{
													"prefix": "/",
													"headers":[{"name":"service","value":"2.0"}]
												},
												"route":{
													"cluster_name":"server_cluster",
													"metadata_match": {
														"filter_metadata": {
															"mosn.lb": {
																 "version":"2.0"
															}
														}
													}
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
				"lb_subset_config": {
					"subset_selectors": [
						["version"]
					]
				},
                                "hosts":[
                                        {
						"address":"127.0.0.1:8080",
						"metadata": {
							"filter_metadata": {
								"mosn.lb": {
									"version":"1.0"
								}
							}
						}
					},
					{
						"address":"127.0.0.1:8081",
						"metadata": {
							"filter_metadata": {
								 "mosn.lb": {
									 "version":"2.0"
							 	}
							}
						}
					}
                                ]
                        }
                ]
        }

}`

func main() {
	lib.Execute(TestSubset)
}

func TestSubset() bool {

	fmt.Println("----- Run http1 subset test ")
	// Init
	mosn := lib.StartMosn(ConfigStr)
	defer mosn.Stop()

	// Server Config
	// If request header contains service_version:1.0, server resposne success
	// If not, server response error (by default)
	srv1 := MakeServer("127.0.0.1:8080", "1.0")
	go srv1.Start()
	// service 2.0
	srv2 := MakeServer("127.0.0.1:8081", "2.0")
	go srv2.Start()
	// Wait Server Start
	time.Sleep(time.Second)

	clientAddrs := []string{
		"127.0.0.1:2045", // client-mosn-mosn-server
		"127.0.0.1:2046", // client-mosn-server
	}
	// test client version 1.0
	for _, addr := range clientAddrs {
		// Client Config
		clt := MakeClient(addr, "1.0")
		// requesy and verify
		for i := 0; i < 5; i++ {
			if !clt.SyncCall() {
				fmt.Printf("client 1.0  request %s is failed\n", addr)
				return false
			}
		}
	}
	// stats verify
	srv1Stats := srv1.ServerStats
	srv2Stats := srv2.ServerStats
	if !(srv1Stats.RequestStats() == uint32(len(clientAddrs)*5) &&
		srv1Stats.ResponseStats()[http.StatusOK] == uint32(len(clientAddrs)*5) &&
		srv2Stats.RequestStats() == 0) {
		fmt.Println("servers request and response is not expected", srv1Stats.RequestStats(), srv2Stats.RequestStats())
		return false
	}
	// test client version 2.0
	for _, addr := range clientAddrs {
		// Client Config
		clt := MakeClient(addr, "2.0")
		// requesy and verify
		for i := 0; i < 5; i++ {
			if !clt.SyncCall() {
				fmt.Printf("client 2.0  request %s is failed\n", addr)
				return false
			}
		}
	}
	if !(srv1Stats.RequestStats() == uint32(len(clientAddrs)*5) &&
		srv1Stats.ResponseStats()[http.StatusOK] == uint32(len(clientAddrs)*5) &&
		srv2Stats.RequestStats() == uint32(len(clientAddrs)*5) &&
		srv2Stats.ResponseStats()[http.StatusOK] == uint32(len(clientAddrs)*5)) {
		fmt.Println("servers request and response is not expected", srv1Stats.RequestStats(), srv2Stats.RequestStats())
		return false
	}
	fmt.Println("----- PASS http1 subset test ")
	return true
}

// Make a mock server, accept header contains service version, response header contains message
func MakeServer(addr string, version string) *testlib_http.MockServer {
	srvConfig := &testlib_http.HTTPServe{
		Configs: map[string]*testlib_http.HTTPResonseConfig{
			"/": {
				ExpectedHeader: map[string][]string{
					"service": []string{version},
				},
				Builder: &testlib_http.ResponseBuilder{
					StatusCode: http.StatusOK,
					Header: map[string]string{
						"message": version,
					},
				},
				ErrorBuidler: testlib_http.DefaultErrorBuilder,
			},
		},
	}
	srv := testlib_http.NewMockServer(addr, srvConfig.Serve)
	return srv
}

// make a mock client, send request header contain version, and want to response header contain message
func MakeClient(addr string, version string) *testlib_http.Client {
	cltVerify := &testlib_http.VerifyConfig{
		ExpectedStatus: http.StatusOK,
		ExpectedHeader: map[string]string{
			"message": version,
		},
	}
	cltConfig := &testlib_http.ClientConfig{
		Addr:          addr,
		MakeRequest:   testlib_http.BuildHTTP1Request,
		RequestMethod: http.MethodGet,
		RequestHeader: map[string]string{
			"service": version,
		},
		Verify: cltVerify.Verify,
	}
	clt := testlib_http.NewClient(cltConfig, 1)
	return clt
}
