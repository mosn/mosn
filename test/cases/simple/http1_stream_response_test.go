//go:build MOSNTest
// +build MOSNTest

package simple

import (
	"context"
	"fmt"
	goHttp "net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
)

func TestHTTP1StreamResponse(t *testing.T) {
	Scenario(t, "http1 stream response", func() {
		var m *mosn.MosnOperator
		server := goHttp.Server{Addr: ":8080"}

		Setup(func() {
			m = mosn.StartMosn(ConfigSimpleHTTPStream)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})

		Case("client-mosn-server", func() {
			var chunk string

			go func() {
				goHttp.HandleFunc("/chunked", func(writer goHttp.ResponseWriter, request *goHttp.Request) {

					flusher, ok := writer.(goHttp.Flusher)
					if !ok {
						goHttp.Error(writer, "Chunked streaming unsupported", goHttp.StatusInternalServerError)
						return
					}

					writer.Header().Set("Content-Type", "text/plain")
					writer.WriteHeader(goHttp.StatusOK)

					// 分三次发送数据块
					for i := 0; i < 3; i++ {
						chunk = fmt.Sprintf("Chunk #%d\n", i+1)
						_, _ = fmt.Fprint(writer, chunk)
						flusher.Flush()
						time.Sleep(1 * time.Second)
					}
				})
				_ = server.ListenAndServe()
			}()

			time.Sleep(time.Second)

			_, _ = goHttp.Get("http://localhost:2046/chunked")

			time.Sleep(500 * time.Millisecond)
			for i := 0; i < 3; i++ {
				assert.Equal(t, chunk, fmt.Sprintf("Chunk #%d\n", i+1))
				fmt.Printf("chunk = %s\n", chunk)
				time.Sleep(1 * time.Second)
			}
		})

		TearDown(func() {
			m.Stop()
			_ = server.Shutdown(context.TODO())
		})
	})
}

const ConfigSimpleHTTPStream = `{
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
