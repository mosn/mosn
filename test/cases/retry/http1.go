package main

import (
	"fmt"
	"net/http"
	"time"

	"sofastack.io/sofa-mosn/test/lib"
	testlib_http "sofastack.io/sofa-mosn/test/lib/http"
)

// client and server's protocol is http1
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

const ConfigStr = `{
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
								"downstream_protocol": "Http1",
								"upstream_protocol": "Http1",
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
											"match":{
												"prefix":"/"
											},
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
								"downstream_protocol": "Http1",
								"upstream_protocol": "Http1",
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
											"match":{
												"prefix":"/"
											},
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
	lib.Execute(TestRetry)
}

func TestRetry() bool {
	fmt.Println("----- RUN http1 retry test ")
	// make mosn config file
	mosn := lib.StartMosn(ConfigStr)
	defer mosn.Stop()

	// start a error server
	srvConfig := &testlib_http.HTTPServe{
		Configs: map[string]*testlib_http.HTTPResonseConfig{
			"/": &testlib_http.HTTPResonseConfig{
				Builder:      testlib_http.DefaultErrorBuilder,
				ErrorBuidler: testlib_http.DefaultErrorBuilder,
			},
		},
	}
	srvError := testlib_http.NewMockServer("127.0.0.1:8080", srvConfig.Serve)
	go srvError.Start()
	defer srvError.Close()
	// start a success (default server)
	srvSuccess := testlib_http.NewMockServer("127.0.0.1:8081", nil)
	go srvSuccess.Start()
	defer srvSuccess.Close()
	// wait server start
	time.Sleep(time.Second)

	// create a simple client
	cfg := testlib_http.CreateSimpleConfig("127.0.0.1:2045")
	VefiyCfg := &testlib_http.VerifyConfig{
		ExpectedStatus: http.StatusOK,
	}
	cfg.Verify = VefiyCfg.Verify
	clt := testlib_http.NewClient(cfg, 1)
	for i := 0; i < 10; i++ {
		if !clt.SyncCall() {
			return false
		}
	}

	// verify
	errStats := srvError.ServerStats
	successStats := srvSuccess.ServerStats
	if !(successStats.RequestStats() == successStats.ResponseStats()[http.StatusOK] &&
		errStats.RequestStats() == errStats.ResponseStats()[http.StatusInternalServerError]) {
		fmt.Println("server response and request does not expected", successStats.RequestStats(), successStats.ResponseStats()[http.StatusOK],
			errStats.RequestStats(), errStats.ResponseStats()[http.StatusInternalServerError])
		return false
	}
	diff := successStats.RequestStats() - errStats.RequestStats()
	if !(diff == 1 || diff == 0) {
		fmt.Printf("success server receive %d requests, error server receive %d requests\n", successStats.RequestStats(), errStats.RequestStats())
		return false
	}
	fmt.Println("----- PASS http1 retry test ")
	return true
}
