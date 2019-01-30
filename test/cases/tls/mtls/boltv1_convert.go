package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/test/lib"
	testlib_sofarpc "github.com/alipay/sofa-mosn/test/lib/sofarpc"
)

/*
If the route path is  client - mosn - mosn - server, the protocol in mosns can be different from client and server.
*/

/*
Verify:
1. client received data is same as server sended
2. server received request count is same as client sended
3. server create only one connection (with mosn2)
*/

const ConfigStrTmpl = `{
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
						"tls_context":{
							"status": true,
							"ca_cert":"-----BEGIN CERTIFICATE-----\nMIIBVzCB/qADAgECAhBsIQij0idqnmDVIxbNRxRCMAoGCCqGSM49BAMCMBIxEDAO\nBgNVBAoTB0FjbWUgQ28wIBcNNzAwMTAxMDAwMDAwWhgPMjA4NDAxMjkxNjAwMDBa\nMBIxEDAOBgNVBAoTB0FjbWUgQ28wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARV\nDG+YT6LzaR5r0Howj4/XxHtr3tJ+llqg9WtTJn0qMy3OEUZRfHdP9iuJ7Ot6rwGF\ni6RXy1PlAurzeFzDqQY8ozQwMjAOBgNVHQ8BAf8EBAMCAqQwDwYDVR0TAQH/BAUw\nAwEB/zAPBgNVHREECDAGhwR/AAABMAoGCCqGSM49BAMCA0gAMEUCIQDt9WA96LJq\nVvKjvGhhTYI9KtbC0X+EIFGba9lsc6+ubAIgTf7UIuFHwSsxIVL9jI5RkNgvCA92\nFoByjq7LS7hLSD8=\n-----END CERTIFICATE-----",
							"cert_chain":"-----BEGIN CERTIFICATE-----\nMIIBdDCCARqgAwIBAgIQbCEIo9Inap5g1SMWzUcUQjAKBggqhkjOPQQDAjASMRAw\nDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYwMDAw\nWjASMRAwDgYDVQQKEwdBY21lIENvMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE\nVQxvmE+i82kea9B6MI+P18R7a97SfpZaoPVrUyZ9KjMtzhFGUXx3T/Yriezreq8B\nhYukV8tT5QLq83hcw6kGPKNQME4wDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQG\nCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMA8GA1UdEQQIMAaHBH8A\nAAEwCgYIKoZIzj0EAwIDSAAwRQIgO9xLIF1AoBsSMU6UgNE7svbelMAdUQgEVIhq\nK3gwoeICIQCDC75Fa3XQL+4PZatS3OfG93XNFyno9koyn5mxLlDAAg==\n-----END CERTIFICATE-----",
							"private_key":"-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEICWksdaVL6sOu33VeohiDuQ3gP8xlQghdc+2FsWPSkrooAoGCCqGSM49\nAwEHoUQDQgAEVQxvmE+i82kea9B6MI+P18R7a97SfpZaoPVrUyZ9KjMtzhFGUXx3\nT/Yriezreq8BhYukV8tT5QLq83hcw6kGPA==\n-----END EC PRIVATE KEY-----",
							"verify_client":true
						},
						"filters": [
							{
								"type": "proxy",
								"config": {
									"downstream_protocol": "%s",
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
				"tls_context": {
					"status": true,
					"server_name": "127.0.0.1",
					"ca_cert":"-----BEGIN CERTIFICATE-----\nMIIBVzCB/qADAgECAhBsIQij0idqnmDVIxbNRxRCMAoGCCqGSM49BAMCMBIxEDAO\nBgNVBAoTB0FjbWUgQ28wIBcNNzAwMTAxMDAwMDAwWhgPMjA4NDAxMjkxNjAwMDBa\nMBIxEDAOBgNVBAoTB0FjbWUgQ28wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARV\nDG+YT6LzaR5r0Howj4/XxHtr3tJ+llqg9WtTJn0qMy3OEUZRfHdP9iuJ7Ot6rwGF\ni6RXy1PlAurzeFzDqQY8ozQwMjAOBgNVHQ8BAf8EBAMCAqQwDwYDVR0TAQH/BAUw\nAwEB/zAPBgNVHREECDAGhwR/AAABMAoGCCqGSM49BAMCA0gAMEUCIQDt9WA96LJq\nVvKjvGhhTYI9KtbC0X+EIFGba9lsc6+ubAIgTf7UIuFHwSsxIVL9jI5RkNgvCA92\nFoByjq7LS7hLSD8=\n-----END CERTIFICATE-----",
					"cert_chain":"-----BEGIN CERTIFICATE-----\nMIIBdDCCARqgAwIBAgIQbCEIo9Inap5g1SMWzUcUQjAKBggqhkjOPQQDAjASMRAw\nDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYwMDAw\nWjASMRAwDgYDVQQKEwdBY21lIENvMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE\nVQxvmE+i82kea9B6MI+P18R7a97SfpZaoPVrUyZ9KjMtzhFGUXx3T/Yriezreq8B\nhYukV8tT5QLq83hcw6kGPKNQME4wDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQG\nCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMA8GA1UdEQQIMAaHBH8A\nAAEwCgYIKoZIzj0EAwIDSAAwRQIgO9xLIF1AoBsSMU6UgNE7svbelMAdUQgEVIhq\nK3gwoeICIQCDC75Fa3XQL+4PZatS3OfG93XNFyno9koyn5mxLlDAAg==\n-----END CERTIFICATE-----",
					"private_key":"-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEICWksdaVL6sOu33VeohiDuQ3gP8xlQghdc+2FsWPSkrooAoGCCqGSM49\nAwEHoUQDQgAEVQxvmE+i82kea9B6MI+P18R7a97SfpZaoPVrUyZ9KjMtzhFGUXx3\nT/Yriezreq8BhYukV8tT5QLq83hcw6kGPA==\n-----END EC PRIVATE KEY-----"
				},


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
	convertList := []string{
		"Http1",
		"Http2",
	}
	for _, proto := range convertList {
		fmt.Println("----- RUN boltv1 -> ", proto, " tls inspector ")
		RunCase(proto)
		fmt.Println("----- PASS boltv1 -> ", proto, " tls inspector ")
	}
}

func RunCase(protocolStr string) {
	// use defer to exit, so the defer close can be executed
	// the first defer will be the last called one
	CasePassed := true
	defer func() {
		if !CasePassed {
			os.Exit(1)
		}
	}()

	// Init
	configStr := fmt.Sprintf(ConfigStrTmpl, protocolStr, protocolStr)
	// start mosn
	mosn := lib.StartMosn(configStr)
	defer mosn.Stop()
	// start a simple boltv1 server
	// the address is same as config (mosn's cluster host address)
	srv := testlib_sofarpc.NewMockServer("127.0.0.1:8080", nil)
	go srv.Start()
	defer srv.Close()
	// wait server start
	time.Sleep(time.Second)

	// create a simple client config, the address is the mosn listener address
	cfg := testlib_sofarpc.CreateSimpleConfig("127.0.0.1:2045")
	// the simple server's response is:
	// Header mosn-test-default: boltv1
	// Content: default-boltv1
	// we will set the client's verify
	VefiyCfg := &testlib_sofarpc.VerifyConfig{
		ExpectedStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
		ExpectedHeader: map[string]string{
			"mosn-test-default": "boltv1",
		},
		ExpectedBody: []byte("default-boltv1"),
	}
	cfg.Verify = VefiyCfg.Verify
	// create only one connection
	clt := testlib_sofarpc.NewClient(cfg, 1)
	// send a request, and verify the result
	if !clt.SyncCall() {
		CasePassed = false
		return
	}
	// Verify the Stats
	connTotal, connActive, connClose := srv.ServerStats.ConnectionStats()
	if !(connTotal == 1 && connActive == 1 && connClose == 0) {
		fmt.Println("server connection is not expected", connTotal, connActive, connClose)
		CasePassed = false
		return
	}
	if !(srv.ServerStats.RequestStats() == 1 && srv.ServerStats.ResponseStats()[sofarpc.RESPONSE_STATUS_SUCCESS] == 1) {
		fmt.Println("server request and response is not expected", srv.ServerStats.RequestStats(), srv.ServerStats.ResponseStats())
		CasePassed = false
		return
	}
}
