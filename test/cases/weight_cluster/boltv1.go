package main

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"sofastack.io/sofa-mosn/test/lib"
	testlib_sofarpc "sofastack.io/sofa-mosn/test/lib/sofarpc"
)

/*
Router config with ClusterWeight
2 clusters named cluster1 and cluster2, each cluster have 1 host
cluster1's weight is 90, cluster2 is 10
*/

/*
Verify:
1. cluster1's request: cluster2's request near 9:1 (allow some range)
2. client received data is same as server sended
3. cluster1 + cluster2 receive request count is same as client sended
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
                                                                        "downstream_protocol": "SofaRpc",
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
                                                                                                 "route":{
													 "weighted_clusters": [
													 	{
															"cluster": {
																"name": "server_cluster1",
																"weight": 90
															}
														},
														{
															"cluster": {
																"name": "server_cluster2",
																"weight": 10
															}
														}
													 ]
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
                                "name": "server_cluster1",
                                "type": "SIMPLE",
                                "lb_type": "LB_RANDOM",
                                "hosts":[
                                        {"address":"127.0.0.1:8080"}
                                ]
                        },
			{
				"name": "server_cluster2",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
 				"hosts":[
                                        {"address":"127.0.0.1:8081"}
                                ]	
			}
                ]
        }

}`

func main() {
	// use defer to exit, so the defer close can be executed
	// the first defer will be the last called one
	CasePassed := true
	defer func() {
		if !CasePassed {
			os.Exit(1)
		}
	}()
	// ignore the client log
	log.InitDefaultLogger("", log.FATAL)

	fmt.Println("----- Run boltv1 cluster weight test ")
	// Init
	// start mosn
	mosn := lib.StartMosn(ConfigStr)
	defer mosn.Stop()
	// cluster1's server host
	srv := testlib_sofarpc.NewMockServer("127.0.0.1:8080", nil)
	go srv.Start()
	// cluster2's server host
	srv2 := testlib_sofarpc.NewMockServer("127.0.0.1:8081", nil)
	go srv2.Start()
	// wait server start
	time.Sleep(time.Second)

	// create a simple client config, the address is the mosn listener address
	clientAddrs := []string{
		"127.0.0.1:2045", // client-mosn-mosn-server
		"127.0.0.1:2046", // client-mosn-server
	}
	var clientRequestCount uint32 = 0
	for _, addr := range clientAddrs {
		cfg := testlib_sofarpc.CreateSimpleConfig(addr)
		// just check response is success
		VefiyCfg := &testlib_sofarpc.VerifyConfig{
			ExpectedStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
		}
		cfg.Verify = VefiyCfg.Verify
		// concurrency requeest the server
		// try to send 100 requests total
		clt := testlib_sofarpc.NewClient(cfg, 4)
		wg := sync.WaitGroup{}
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 25; j++ {
					if !clt.SyncCall() {
						fmt.Printf("client request %s is failed\n", addr)
						CasePassed = false
						return
					}
				}
			}()
		}
		wg.Wait()
		//Verify client
		if !CasePassed {
			return
		}
		clientRequestCount += clt.Stats.RequestStats()
	}

	// Verify Stats
	stats1 := srv.ServerStats
	stats2 := srv2.ServerStats
	var requestTotal uint32 = stats1.RequestStats() + stats2.RequestStats()
	var responseTotal uint32 = stats1.ResponseStats()[sofarpc.RESPONSE_STATUS_SUCCESS] + stats2.ResponseStats()[sofarpc.RESPONSE_STATUS_SUCCESS]
	if requestTotal != clientRequestCount || responseTotal != clientRequestCount {
		fmt.Printf("server request total %d not equal client request total %d, success repsonse is %d\n", requestTotal, clientRequestCount, responseTotal)
		CasePassed = false
		return
	}
	thres := 0.05
	if math.Abs(float64(stats1.RequestStats())/float64(requestTotal)-0.9) > thres {
		fmt.Printf("cluster1 expected contains 90% request, but got %d , total request is %d\n", stats1.RequestStats(), requestTotal)
		CasePassed = false
		return
	}
	if math.Abs(float64(stats2.RequestStats())/float64(requestTotal)-0.1) > thres {
		fmt.Printf("cluster2 expected contains 10% request, but got %d , total request is %d\n", stats2.RequestStats(), requestTotal)
		CasePassed = false
		return
	}
	fmt.Println("---- PASS boltv1 cluster weight test ")

}
