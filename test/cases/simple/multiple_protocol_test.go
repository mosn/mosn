//go:build MOSNTest
// +build MOSNTest

package simple

import (
	"fmt"
	"testing"
	"time"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/http"
	"mosn.io/mosn/test/lib/xprotocol/boltv1"
	"mosn.io/mosn/test/lib/xprotocol/boltv2"
)

const MockBoltV2ServerConfig = `{
	"protocol":"boltv2",
	"config": {
		"address": "127.0.0.1:8080"
	}
}`

const MockHttp2ServerConfig = `{
	"protocol":"Http2",
	"config": {
		"address": "127.0.0.1:8080"
	}
}`

func TestOneProtocolConfig(t *testing.T) {
	Scenario(t, "downstream protocol config to bolt", func() {
		lib.InitMosn(CreateConfigByProtocol("bolt"), lib.CreateConfig(MockBoltServerConfig))
		Case("call", func() {
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)
			Verify(stats.ConnectionClosed(), Equal, 0)
		})
	})
	Scenario(t, "downstream protocol config to boltv2", func() {
		lib.InitMosn(CreateConfigByProtocol("boltv2"), lib.CreateConfig(MockBoltV2ServerConfig))
		Case("call", func() {
			client := lib.CreateClient("boltv2", &boltv2.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &boltv2.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)
			Verify(stats.ConnectionClosed(), Equal, 0)
		})
	})
	Scenario(t, "downstream protocol config to http1", func() {
		lib.InitMosn(CreateConfigByProtocol("Http1"), lib.CreateConfig(MockHttpServerConfig))
		Case("call", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)
			Verify(stats.ConnectionClosed(), Equal, 0)
		})
	})
	Scenario(t, "downstream protocol config to http2", func() {
		lib.InitMosn(CreateConfigByProtocol("Http2"), lib.CreateConfig(MockHttp2ServerConfig))
		Case("call", func() {
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)
			Verify(stats.ConnectionClosed(), Equal, 0)
		})
	})
}

func TestAutoProtocol(t *testing.T) {
	Scenario(t, "downstream protocol config as auto", func() {
		lib.InitMosn(CreateConfigByProtocol("Auto"))
		Case("call bolt", func() {
			sc := lib.CreateConfig(MockBoltServerConfig)
			srv := lib.CreateServer(sc.Protocol, sc.Config)
			defer srv.Stop()
			go srv.Start()
			time.Sleep(time.Second) // wait server start
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.ConnectionClosed(), Equal, 0)
		})
		Case("call boltv2", func() {
			sc := lib.CreateConfig(MockBoltV2ServerConfig)
			srv := lib.CreateServer(sc.Protocol, sc.Config)
			defer srv.Stop()
			go srv.Start()
			time.Sleep(time.Second) // wait server start
			client := lib.CreateClient("boltv2", &boltv2.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &boltv2.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.ConnectionClosed(), Equal, 0)

		})
		Case("call http", func() {
			sc := lib.CreateConfig(MockHttpServerConfig)
			srv := lib.CreateServer(sc.Protocol, sc.Config)
			defer srv.Stop()
			go srv.Start()
			time.Sleep(time.Second) // wait server start
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.ConnectionClosed(), Equal, 0)

		})
		Case("call http2", func() {
			sc := lib.CreateConfig(MockHttp2ServerConfig)
			srv := lib.CreateServer(sc.Protocol, sc.Config)
			defer srv.Stop()
			go srv.Start()
			time.Sleep(time.Second) // wait server start
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.ConnectionClosed(), Equal, 0)
		})

	})
}

func TestSofaRPCAutoProtocol(t *testing.T) {
	Scenario(t, "downstream protocol config as sofarpc", func() {
		lib.InitMosn(CreateConfigByProtocol("bolt,boltv2"))
		Case("call bolt", func() {
			sc := lib.CreateConfig(MockBoltServerConfig)
			srv := lib.CreateServer(sc.Protocol, sc.Config)
			defer srv.Stop()
			go srv.Start()
			time.Sleep(time.Second) // wait server start
			client := lib.CreateClient("bolt", &boltv1.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &boltv1.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.ConnectionClosed(), Equal, 0)
		})
		Case("call boltv2", func() {
			sc := lib.CreateConfig(MockBoltV2ServerConfig)
			srv := lib.CreateServer(sc.Protocol, sc.Config)
			defer srv.Stop()
			go srv.Start()
			time.Sleep(time.Second) // wait server start
			client := lib.CreateClient("boltv2", &boltv2.BoltClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &boltv2.VerifyConfig{
					ExpectedStatusCode: 0,
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.ConnectionClosed(), Equal, 0)

		})
		Case("call http", func() {
			sc := lib.CreateConfig(MockHttpServerConfig)
			srv := lib.CreateServer(sc.Protocol, sc.Config)
			defer srv.Stop()
			go srv.Start()
			time.Sleep(time.Second) // wait server start
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
			})
			// cannot request for http, got a connection closed error
			Verify(client.SyncCall(), Equal, false)
			stats := client.Stats()
			Verify(stats.ConnectionClosed(), Equal, 1)

		})

		Case("call http2", func() {
			sc := lib.CreateConfig(MockHttp2ServerConfig)
			srv := lib.CreateServer(sc.Protocol, sc.Config)
			defer srv.Stop()
			go srv.Start()
			time.Sleep(time.Second) // wait server start
			client := lib.CreateClient("Http2", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
			})
			// cannot request for http2, got a connection closed error
			Verify(client.SyncCall(), Equal, false)
			stats := client.Stats()
			Verify(stats.ConnectionClosed(), Equal, 1)
		})

	})
}

func CreateConfigByProtocol(protos string) string {
	return fmt.Sprintf(ConfigSimpleMultipleTmpl, protos)
}

const ConfigSimpleMultipleTmpl = `{
	 "servers":[
	 	{
			"default_log_path":"stdout",
			"default_log_level": "ERROR",
			"routers":[
				{
					"router_config_name":"router_to_server",
					"virtual_hosts":[{
						"name":"server_hosts",
						"domains": ["*"],
						"routers": [
							{
								"match":{},
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
									"downstream_protocol":"%s",
									"router_config_name":"router_to_server"
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
