//go:build Skip
// +build Skip

package tproxy

import (
	"io/ioutil"
	"net/http"
	"os/exec"
	"testing"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
)

func request(t *testing.T, url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	return b
}

func testEnvScript(path string) error {
	runCmd := exec.Command("sudo", "/bin/bash", path)
	_, err := runCmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

// TODO Execute this test using a docker image containing iptables and sudo
func TestSimpleTProxy(t *testing.T) {

	err := testEnvScript("./setup.sh")

	defer testEnvScript("./cleanup.sh")

	if err != nil {
		t.Fatal(err)
	}

	Scenario(t, "test transparent proxy egress", func() {
		lib.InitMosn(ConfigTProxy)
		Case("general listener", func() {
			Equal(t, request(t, "http://2.2.2.2:80"), []byte("this is general_server"))
		})
		Case("proxy listener", func() {
			Equal(t, request(t, "http://2.2.2.2:12345"), []byte("this is tproxy_server"))
		})
	})
}

const ConfigTProxy = `{
	"servers":[
		{
			"default_log_path":"stdout",
			"routers": [
				{
                    "router_config_name": "tproxy_router",
                    "virtual_hosts": [
						{
							"name": "tproxy_server",
							"domains": [
								"*"
							],
							"routers": [
								{
									"direct_response": {
										"status": 200,
										"body": "this is tproxy_server"
									}
								}
							]
						}
                    ]
                },
				{
                    "router_config_name": "general_router",
                    "virtual_hosts": [
						{
							"name": "general_server",
							"domains": [
								"*"
							],
							"routers": [
								{
									"direct_response": {
										"status": 200,
										"body": "this is general_server"
									}
								}
							]
						}
                    ]
                }
			],
			"listeners":[
				{
					"name":"tproxy_listener",
					"address": "0.0.0.0:16000",
                    "bind_port": true,
					"use_original_dst": "tproxy",
					"listener_filters": [
						{
							"type": "original_dst",
							"config":{
								"type": "tproxy"
							}
						}
					],
					"filter_chains": [
                        {
                            "filters": [
                                {
                                    "type": "proxy",
                                    "config": {
                                        "downstream_protocol": "Auto",
                                        "upstream_protocol": "Auto",
                                        "router_config_name": "tproxy_router"
                                    }
                                }
                            ]
                        }
                    ]
				},
				{
					"name": "general_listener",
                    "address": "0.0.0.0:80",
                    "bind_port": false,
                    "filter_chains": [
                        {
                            "filters": [
                                {
                                    "type": "proxy",
                                    "config": {
                                        "downstream_protocol": "Auto",
                                        "upstream_protocol": "Auto",
                                        "router_config_name": "general_router"
                                    }
                                }
                            ]
                        }
                    ]
				}
			]
		}
	],
	"admin": {
		"address": {
			"socket_address": {
				"address": "0.0.0.0",
				"port_value": 34901
			}
		}
	}
}
`
