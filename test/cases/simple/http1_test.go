//go:build MOSNTest
// +build MOSNTest

package simple

import (
	"context"
	"crypto/tls"
	"net"
	goHttp "net/http"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"mosn.io/mosn/pkg/module/http2/h2c"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/http"
	"mosn.io/mosn/test/lib/mosn"
)

var urlTestCases = []struct {
	inputURL        string
	wantPATH        string
	wantQueryString string
}{
	{
		"/",
		"/",
		"",
	},
	{
		"/abc",
		"/abc",
		"",
	},
	{
		"/aa%20bb",
		"/aa bb",
		"",
	},
	{
		"/aa%20bb%26cc",
		"/aa bb&cc",
		"",
	},
	{
		"/abc+def",
		"/abc+def",
		"",
	},
	{
		"/10%25",
		"/10%",
		"",
	},
	{
		"/1%41",
		"/1A",
		"",
	},
	{
		"/1%41%42%43",
		"/1ABC",
		"",
	},
	{
		"/%4a",
		"/J",
		"",
	},
	{
		"/%6F",
		"/o",
		"",
	},
	{
		"/%20%3F&=%23+%25%21%3C%3E%23%22%7B%7D%7C%5C%5E%5B%5D%60%E2%98%BA%09:%2F@$%27%28%29%2A%2C%3B",
		"/ ?&=#+%!<>#\"{}|\\^[]`☺\t:/@$'()*,;",
		"",
	},
	{
		"/home/;some/sample",
		"/home/;some/sample",
		"",
	},
	{
		"/aa?bb=cc",
		"/aa",
		"bb=cc",
	},
	{
		"/?",
		"/",
		"",
	},
	{
		"/?foo=bar?",
		"/",
		"foo=bar?",
	},
	{
		"/?q=go+language",
		"/",
		"q=go+language",
	},
	{
		"/?q=go%20language",
		"/",
		"q=go%20language",
	},
	{
		"/aa%20bb?q=go%20language",
		"/aa bb",
		"q=go%20language",
	},
	{
		"/aa?+%3F%26%3D%23%2B%25%21%3C%3E%23%22%7B%7D%7C%5C%5E%5B%5D%60%E2%98%BA%09%3A%2F%40%24%27%28%29%2A%2C%3B",
		"/aa",
		"+%3F%26%3D%23%2B%25%21%3C%3E%23%22%7B%7D%7C%5C%5E%5B%5D%60%E2%98%BA%09%3A%2F%40%24%27%28%29%2A%2C%3B",
	},
}

func TestHTTP1UrlPathQuery(t *testing.T) {
	Scenario(t, "http1 url path and query string", func() {
		var m *mosn.MosnOperator
		server := goHttp.Server{Addr: ":8080"}
		Setup(func() {
			m = mosn.StartMosn(ConfigSimpleHTTP1)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("client-mosn-server", func() {
			var serverPath, serverQueryString string

			go func() {
				goHttp.HandleFunc("/", func(writer goHttp.ResponseWriter, request *goHttp.Request) {
					serverPath = request.URL.Path
					serverQueryString = request.URL.RawQuery
				})
				_ = server.ListenAndServe()
			}()

			time.Sleep(time.Second)

			for _, tc := range urlTestCases {
				serverPath, serverQueryString = "", ""

				_, err := goHttp.Get("http://localhost:2046" + tc.inputURL)

				Verify(err, Equal, nil)
				Verify(serverPath, Equal, tc.wantPATH)
				Verify(serverQueryString, Equal, tc.wantQueryString)
			}
		})
		TearDown(func() {
			m.Stop()
			_ = server.Shutdown(context.TODO())
		})
	})
}

func TestHTTP2UrlPathQuery(t *testing.T) {
	Scenario(t, "http2 url path and query string", func() {
		var m *mosn.MosnOperator
		var server *goHttp.Server
		Setup(func() {
			m = mosn.StartMosn(ConfigSimpleHTTP2)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("client-mosn-server", func() {
			var serverPath, serverQueryString string

			go func() {
				h2s := &http2.Server{}
				handler := goHttp.HandlerFunc(func(w goHttp.ResponseWriter, r *goHttp.Request) {
					serverPath = r.URL.Path
					serverQueryString = r.URL.RawQuery
				})
				server = &goHttp.Server{
					Addr:    "0.0.0.0:8080",
					Handler: h2c.NewHandler(handler, h2s),
				}
				_ = server.ListenAndServe()
			}()

			time.Sleep(time.Second)

			for _, tc := range urlTestCases {
				serverPath, serverQueryString = "", ""

				client := goHttp.Client{
					Transport: &http2.Transport{
						AllowHTTP: true,
						DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
							return net.Dial(network, addr)
						},
					},
				}
				_, err := client.Get("http://localhost:2046" + tc.inputURL)

				Verify(err, Equal, nil)
				Verify(serverPath, Equal, tc.wantPATH)
				Verify(serverQueryString, Equal, tc.wantQueryString)
			}
		})
		TearDown(func() {
			m.Stop()
			_ = server.Shutdown(context.TODO())
		})
	})
}

func TestSimpleHTTP1(t *testing.T) {
	Scenario(t, "simple http1 proxy used mosn", func() {
		// servers is invalid in `Case`
		_, servers := lib.InitMosn(ConfigSimpleHTTP1, lib.CreateConfig(MockHttpServerConfig))
		Case("client-mosn-mosn-server", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2045",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedHeader: map[string][]string{
						"mosn-test-default": []string{"http1"},
					},
					ExpectedBody: []byte("default-http1"),
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)
		})
		Case("client-mosn-server", func() {
			client := lib.CreateClient("Http1", &http.HttpClientConfig{
				TargetAddr: "127.0.0.1:2046",
				Verify: &http.VerifyConfig{
					ExpectedStatusCode: 200,
					ExpectedHeader: map[string][]string{
						"mosn-test-default": []string{"http1"},
					},
					ExpectedBody: []byte("default-http1"),
				},
			})
			Verify(client.SyncCall(), Equal, true)
			stats := client.Stats()
			Verify(stats.Requests(), Equal, 1)
			Verify(stats.ExpectedResponseCount(), Equal, 1)

		})
		Case("server-verify", func() {
			srv := servers[0]
			stats := srv.Stats()
			Verify(stats.ConnectionTotal(), Equal, 1)
			Verify(stats.ConnectionActive(), Equal, 1)
			Verify(stats.ConnectionClosed(), Equal, 0)
			Verify(stats.Requests(), Equal, 2)

		})
	})
}

const MockHttpServerConfig = `{
	"protocol":"Http1",
	"config": {
		"address": "127.0.0.1:8080"
	}
}`

const ConfigSimpleHTTP1 = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "ERROR",
                        "routers": [
                                {
                                        "router_config_name":"router_to_mosn",
                                        "virtual_hosts":[{
                                                "name":"mosn_hosts",
                                                "domains": ["*"],
                                                "routers": [
                                                        {
                                                                "match":{"prefix":"/"},
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
                                                                "match":{"prefix":"/"},
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
                                                                        "downstream_protocol": "Http1",
                                                                        "upstream_protocol": "Http1",
                                                                        "router_config_name":"router_to_mosn"
                                                                }
                                                        }
                                                ]
                                        }]
                                },
                                {
                                        "address":"127.0.0.1:2046",
                                        "bind_port": true,
                                        "filter_chains": [{
                                                "filters": [
                                                        {
                                                                "type": "proxy",
                                                                "config": {
                                                                        "downstream_protocol": "Http1",
                                                                        "upstream_protocol": "Http1",
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

const ConfigSimpleHTTP2 = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "ERROR",
                        "routers": [
                                {
                                        "router_config_name":"router_to_server",
                                        "virtual_hosts":[{
                                                "name":"server_hosts",
                                                "domains": ["*"],
                                                "routers": [
                                                        {
                                                                "match":{"prefix":"/"},
                                                                "route":{"cluster_name":"server_cluster"}
                                                        }
                                                ]
                                        }]
                                }
                        ],
                        "listeners":[
                                {
                                        "address":"127.0.0.1:2046",
                                        "bind_port": true,
                                        "filter_chains": [{
                                                "filters": [
                                                        {
                                                                "type": "proxy",
                                                                "config": {
                                                                        "downstream_protocol": "Http2",
                                                                        "upstream_protocol": "Http2",
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
