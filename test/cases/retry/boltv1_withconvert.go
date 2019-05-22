package main

import (
	"fmt"
	"os"
	"time"

	"sofastack.io/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"sofastack.io/sofa-mosn/test/lib"
	testlib_sofarpc "sofastack.io/sofa-mosn/test/lib/sofarpc"
)

// client and server's protocol is boltv1
// mosn-mosn[s] support boltv1(sofa rpc)\http\http2
// mosn: 2045 is the client request, will send to mosn1 or mosn2
// mosn1: 2046. send request to server:8080, response is error
// mosn2: 2047. send request to server:8081, response is success
// send 2 requests, make sure mosn1/mosn2 receeive request
// Verify:
// 1. client received response must be ok
// 2. requests[success] - reuqests[error] < 1.
// If the first request is route to "error", the requests[success] = requests[error]
// If the first request is route to "success", the requests[success]-1 = requests[error]
// TODO: mosn retry metrics

const ConfigStrTmpl = `{
	"servers":[{
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
											"route":{
												"cluster_name":"mosn_cluster",
												"retry_policy": {
													"retry_on": true,
													"num_retries": 3
												}
											}
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
				 "log_level": "FATAL",
				 "filter_chains": [{
					 "filters": [
					 	{
							"type": "proxy",
							"config": {
								"downstream_protocol": "%s",
								"upstream_protocol": "SofaRpc",
								"router_config_name":"router_to_error"
							}
						},
						{
							"type": "connection_manager",
							"config": {
								"router_config_name":"router_to_error",
								"virtual_hosts":[{
									"name":"error_hosts",
									"domains": ["*"],
									"routers": [
										{
											"match":{"headers":[{"name":"service","value":".*"}]},
											"route":{"cluster_name":"error_cluster"}
										}
									]
								}]
							}
						}
					 ]
				 }]
			},
			{
				 "address":"127.0.0.1:2047",
				 "bind_port": true,
				 "log_path": "stdout",
				 "log_level": "FATAL",
				 "filter_chains": [{
					 "filters": [
					 	{
							"type": "proxy",
							"config": {
								"downstream_protocol": "%s",
								"upstream_protocol": "SofaRpc",
								"router_config_name":"router_to_success"
							}
						},
						{
							"type": "connection_manager",
							"config": {
								"router_config_name":"router_to_success",
								"virtual_hosts":[{
									"name":"success_hosts",
									"domains": ["*"],
									"routers": [
										{
											"match":{"headers":[{"name":"service","value":".*"}]},
											"route":{"cluster_name":"success_cluster"}
										}
									]
								}]
							}
						}
					 ]
				 }]
			}

		 ]
	}],
	"cluster_manager":{
		"clusters":[
			{
				"name": "mosn_cluster",
				"type": "SIMPLE",
				"lb_type": "LB_ROUNDROBIN",
				"hosts":[
					{"address":"127.0.0.1:2046"},
					{"address":"127.0.0.1:2047"}
				]
			},
			{
				"name": "error_cluster",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
				"hosts":[
					{"address":"127.0.0.1:8080"}
				]
			},
			{
				"name": "success_cluster",
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
	protoList := []string{
		"SofaRpc",
		// TODO: currently, do not support sofa-http convert with retry
		// "Http1",
		//	"Http2",
	}
	for _, proto := range protoList {
		fmt.Println("----- RUN boltv1 -> ", proto, " retry test ")
		if !RunCase(proto) {
			os.Exit(1)
		}
		fmt.Println("----- PASS boltv1 -> ", proto, " retry test ")
	}
}

func RunCase(proto string) bool {
	// make mosn config file
	configStr := fmt.Sprintf(ConfigStrTmpl, proto, proto, proto)
	mosn := lib.StartMosn(configStr)
	defer mosn.Stop()

	// start a error server
	srvConfig := &testlib_sofarpc.BoltV1Serve{
		Configs: []*testlib_sofarpc.BoltV1ReponseConfig{
			{
				Builder: &testlib_sofarpc.BoltV1ResponseBuilder{
					Status: sofarpc.RESPONSE_STATUS_ERROR,
				},
			},
		},
	}
	srvError := testlib_sofarpc.NewMockServer("127.0.0.1:8080", srvConfig.Serve)
	go srvError.Start()
	defer srvError.Close()
	// start a success (default server)
	srvSuccess := testlib_sofarpc.NewMockServer("127.0.0.1:8081", nil)
	go srvSuccess.Start()
	defer srvSuccess.Close()
	// wait server start
	time.Sleep(time.Second)

	// create a simple client
	cfg := testlib_sofarpc.CreateSimpleConfig("127.0.0.1:2045")
	VefiyCfg := &testlib_sofarpc.VerifyConfig{
		ExpectedStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
	}
	cfg.Verify = VefiyCfg.Verify
	// TODO:
	// Fix LB concurrency
	// a stream get a lb, use it to choose host
	// but another stream also use this lb to choose host
	// when we do retry, a stream may be choose a same host when concurrency
	/*
		passed := true
		clt := testlib_sofarpc.NewClient(cfg, 3)
		wg := sync.WaitGroup{}
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 4; j++ {
					if !clt.SyncCall() {
						passed = false
						return
					}
				}
			}()
		}
		wg.Wait()
		if !passed {
			return passed
		}
	*/
	clt := testlib_sofarpc.NewClient(cfg, 1)
	for i := 0; i < 10; i++ {
		if !clt.SyncCall() {
			return false
		}
	}

	// verify
	errStats := srvError.ServerStats
	successStats := srvSuccess.ServerStats
	if !(successStats.RequestStats() == successStats.ResponseStats()[sofarpc.RESPONSE_STATUS_SUCCESS] &&
		errStats.RequestStats() == errStats.ResponseStats()[sofarpc.RESPONSE_STATUS_ERROR]) {
		fmt.Println("server response and request does not expected", successStats.RequestStats(), successStats.ResponseStats()[sofarpc.RESPONSE_STATUS_SUCCESS],
			errStats.RequestStats(), errStats.ResponseStats()[sofarpc.RESPONSE_STATUS_ERROR])
		return false
	}
	diff := successStats.RequestStats() - errStats.RequestStats()
	if !(diff == 1 || diff == 0) {
		fmt.Printf("success server receive %d requests, error server receive %d requests\n", successStats.RequestStats(), errStats.RequestStats())
		return false
	}
	return true
}
