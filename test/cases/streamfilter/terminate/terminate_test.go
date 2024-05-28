//go:build MOSNTest
// +build MOSNTest

package terminate

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/xprotocol/boltv1"
)

func TriggerTerminate(t *testing.T) {
	resp, err := http.Get("http://127.0.0.1:12345")
	if err != nil {
		t.Fatalf("request terminate failed: %v", err)
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	t.Logf("%s, %s", time.Now().String(), string(b))
}

func TestTerminateRequest(t *testing.T) {
	Scenario(t, "test terminate request by mock terminate filter", func() {
		_, _ = lib.InitMosn(ConfigTerminateRequest)
		Case("request and receive a connection failed becasue no server listened", func() {
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Request: &boltv1.RequestConfig{
					Header: map[string]string{
						"service": "test",
					},
					Timeout: 5 * time.Second,
				},
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 6, // expected got a ResponseStatusNoProcessor
				},
			})
			Verify(client.SyncCall(), Equal, true)
		})
		Case("request and trigger terminate", func() {
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Request: &boltv1.RequestConfig{
					Header: map[string]string{
						"service": "test",
					},
					Timeout: 5 * time.Second,
				},
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 7, // expected got a timeout response
				},
			})
			ch := make(chan struct{})
			go func() {
				Verify(client.SyncCall(), Equal, true)
				close(ch)
			}()
			time.Sleep(time.Second) // wait request send
			// Trigger Terminate
			TriggerTerminate(t)
			select {
			case <-ch: // wait sync call
			case <-time.After(6 * time.Second):
				t.Fatalf("wait sync call timeout")
			}
		})
	})
}

const ConfigTerminateRequest = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "ERROR",
                        "routers":[
                                {
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
                                },
                                {
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
                        ],
                        "listeners":[
                                {
                                        "address":"127.0.0.1:2045",
                                        "bind_port": true,
                                        "filter_chains": [{
                                                "filters": [
                                                        {
                                                                "type": "proxy",
                                                                "config": {
                                                                        "downstream_protocol": "X",
                                                                        "upstream_protocol": "X",
                                                                        "extend_config": {
                                                                                "sub_protocol": "bolt"
                                                                        },
                                                                        "router_config_name":"router_to_mosn"
                                                                }
                                                        }
                                                ]
                                        }],
					"stream_filters": [
						{"type":"mock_terminate"}
					]
                                },
                                {
                                        "address":"127.0.0.1:2046",
                                        "bind_port": true,
                                        "filter_chains": [{
                                                "filters": [
                                                        {
                                                                "type": "proxy",
                                                                "config": {
                                                                        "downstream_protocol": "X",
                                                                        "upstream_protocol": "X",
                                                                        "extend_config": {
                                                                                "sub_protocol": "bolt"
                                                                        },
                                                                        "router_config_name":"router_to_server"
                                                                }
                                                        }
                                                ]
                                        }],
					"stream_filters": [
						{
							"type": "fault",
							"config": {
								"delay": {
									"percentage": 100,
									"fixed_delay": "2s"
								}
							}
						}
					]
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
