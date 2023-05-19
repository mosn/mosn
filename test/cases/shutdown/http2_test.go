//go:build MOSNTest
// +build MOSNTest

package shutdown

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net"
	goHttp "net/http"
	"testing"
	"time"

	"mosn.io/mosn/pkg/log"

	"golang.org/x/net/http2"
	"mosn.io/mosn/pkg/module/http2/h2c"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
)

func TestHTTP2GracefulStop(t *testing.T) {
	Scenario(t, "graceful stop", func() {
		var m *mosn.MosnOperator
		var server *goHttp.Server
		Setup(func() {
			m = mosn.StartMosn(ConfigSimpleHTTP2)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("client-mosn-server", func() {
			go func() {
				h2s := &http2.Server{}
				handler := goHttp.HandlerFunc(func(w goHttp.ResponseWriter, r *goHttp.Request) {
					var body []byte
					var err error
					if body, err = ioutil.ReadAll(r.Body); err != nil {
						t.Fatalf("error request body %v", err)
					}
					w.Header().Set("Content-Type", "text/plain")
					time.Sleep(time.Millisecond * 500)
					w.Write(body)
				})
				server = &goHttp.Server{
					Addr:    "0.0.0.0:8080",
					Handler: h2c.NewHandler(handler, h2s),
				}
				_ = server.ListenAndServe()
			}()

			time.Sleep(time.Second)

			testcases := []struct {
				reqBody []byte
			}{
				{
					reqBody: []byte("test-req-body"),
				},
			}

			for _, tc := range testcases {
				client := goHttp.Client{
					Transport: &http2.Transport{
						AllowHTTP: true,
						DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
							return net.Dial(network, addr)
						},
					},
				}
				// 1. simple
				var start time.Time
				start = time.Now()
				resp, err := client.Post("http://localhost:2046", "text/plain", bytes.NewReader(tc.reqBody))
				log.DefaultLogger.Infof("client.Post cost %v", time.Since(start))
				Verify(err, Equal, nil)
				respBody, err := ioutil.ReadAll(resp.Body)
				Verify(err, Equal, nil)
				Verify(len(respBody), Equal, len(tc.reqBody))

				// 2. graceful stop after send request and before received the response
				go func() {
					time.Sleep(time.Millisecond * 100)
					m.GracefulStop()
				}()
				start = time.Now()
				resp, err = client.Post("http://localhost:2046", "text/plain", bytes.NewReader(tc.reqBody))
				log.DefaultLogger.Infof("client.Post cost %v", time.Since(start))
				Verify(err, Equal, nil)
				respBody, err = ioutil.ReadAll(resp.Body)
				Verify(err, Equal, nil)
				Verify(len(respBody), Equal, len(tc.reqBody))

				// 3. after graceful stop, make sure the server is closed
				start = time.Now()
				resp, err = client.Post("http://localhost:2046", "text/plain", bytes.NewReader(tc.reqBody))
				log.DefaultLogger.Infof("client.Post cost %v", time.Since(start))
				Verify(err, NotEqual, nil)
			}
		})
		TearDown(func() {
			m.Stop()
			_ = server.Shutdown(context.TODO())
		})
	})
}

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
