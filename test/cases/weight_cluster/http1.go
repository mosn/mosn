package main

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/test/lib"
	testlib_http "sofastack.io/sofa-mosn/test/lib/http"
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
	lib.Execute(TestWeightCluster)
}

func TestWeightCluster() bool {
	// ignore the client log
	log.InitDefaultLogger("", log.FATAL)

	fmt.Println("----- Run http1 cluster weight test ")
	// Init
	// start mosn
	mosn := lib.StartMosn(ConfigStr)
	defer mosn.Stop()
	// cluster1's server host
	srv := testlib_http.NewMockServer("127.0.0.1:8080", nil)
	go srv.Start()
	// cluster2's server host
	srv2 := testlib_http.NewMockServer("127.0.0.1:8081", nil)
	go srv2.Start()
	// wait server start
	time.Sleep(time.Second)

	// create a simple client config, the address is the mosn listener address
	clientAddrs := []string{
		"127.0.0.1:2045", // client-mosn-mosn-server
		"127.0.0.1:2046", // client-mosn-server
	}
	var clientRequestCount uint32 = 0
	var passed = true
	for _, addr := range clientAddrs {
		cfg := testlib_http.CreateSimpleConfig(addr)
		// just check response is success
		VefiyCfg := &testlib_http.VerifyConfig{
			ExpectedStatus: http.StatusOK,
		}
		cfg.Verify = VefiyCfg.Verify
		// concurrency requeest the server
		// try to send 100 requests total
		clt := testlib_http.NewClient(cfg, 4)
		wg := sync.WaitGroup{}
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 25; j++ {
					if !clt.SyncCall() {
						fmt.Printf("client request %s is failed\n", addr)
						passed = false
						return
					}
				}
			}()
		}
		wg.Wait()
		//Verify client
		if !passed {
			return false
		}
		clientRequestCount += clt.Stats.RequestStats()
	}

	// Verify Stats
	stats1 := srv.ServerStats
	stats2 := srv2.ServerStats
	var requestTotal uint32 = stats1.RequestStats() + stats2.RequestStats()
	var responseTotal uint32 = stats1.ResponseStats()[http.StatusOK] + stats2.ResponseStats()[http.StatusOK]
	if requestTotal != clientRequestCount || responseTotal != clientRequestCount {
		fmt.Printf("server request total %d not equal client request total %d, success repsonse is %d\n", requestTotal, clientRequestCount, responseTotal)
		return false
	}
	thres := 0.05
	if math.Abs(float64(stats1.RequestStats())/float64(requestTotal)-0.9) > thres {
		fmt.Printf("cluster1 expected contains 90% request, but got %d , total request is %d\n", stats1.RequestStats(), requestTotal)
		return false
	}
	if math.Abs(float64(stats2.RequestStats())/float64(requestTotal)-0.1) > thres {
		fmt.Printf("cluster2 expected contains 10% request, but got %d , total request is %d\n", stats2.RequestStats(), requestTotal)
		return false
	}
	fmt.Println("---- PASS http1 cluster weight test ")
	return true
}
