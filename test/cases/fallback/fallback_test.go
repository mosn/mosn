// +build MOSNTest

package fallback

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"bou.ke/monkey"
	"mosn.io/api"
	"mosn.io/mosn/pkg/filter/listener/originaldst"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
)

type mockListenerFilterChainFactory struct{}

func (filter *mockListenerFilterChainFactory) OnAccept(cb api.ListenerFilterChainFactoryCallbacks) api.FilterStatus {
	if !cb.GetUseOriginalDst() {
		return api.Continue
	}
	ips := fmt.Sprintf("%d.%d.%d.%d", 127, 0, 0, 1)
	cb.SetOriginalAddr(ips, 9092)
	cb.UseOriginalDst(cb.GetOriContext())
	return api.Stop
}

func TestProxyFallback(t *testing.T) {
	Scenario(t, "proxy fallback to tcp-proxy", func() {
		var m *mosn.MosnOperator
		h1Server := http.Server{Addr: ":9091"}
		var tcpListener net.Listener
		h1ReqCount, tcpReqCount := 0, 0
		tcpReadBytes := 0

		Setup(func() {
			m = mosn.StartMosn(ConfigSimpleHTTP1)
			Verify(m, NotNil)

			go func() {
				http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
					h1ReqCount++
					writer.WriteHeader(http.StatusOK)
				})
				_ = h1Server.ListenAndServe()
			}()

			go func() {
				tcpListener, _ = net.Listen("tcp", "127.0.0.1:9092")
				Verify(tcpListener, NotNil)
				for {
					conn, err := tcpListener.Accept()
					Verify(err, Equal, nil)
					tcpReqCount++
					go func() {
						buf := make([]byte, 1024)
						tcpReadBytes, _ = conn.Read(buf)
					}()
				}
			}()

			time.Sleep(2 * time.Second) // wait mosn start

			monkey.Patch(originaldst.NewOriginalDst, func() api.ListenerFilterChainFactory {
				return &mockListenerFilterChainFactory{}
			})
		})

		Case("client-mosn-server", func() {
			// http req should be processed by proxy
			_, err := http.Get("http://localhost:2046/")
			Verify(err, Equal, nil)
			Verify(h1ReqCount, Equal, 1)

			// unknown protocol should fallback to tcp-proxy
			c, err := net.Dial("tcp", "127.0.0.1:2046")
			Verify(c, NotNil)
			Verify(err, Equal, nil)
			Verify(tcpReqCount, Equal, 1)

			tcpBytes := []byte("helloWorld")
			_, err = c.Write(tcpBytes)
			Verify(err, Equal, nil)
			time.Sleep(time.Second)
			Verify(tcpReadBytes, Equal, len(tcpBytes))

			// the first req of a connection does not wait for more data
			c, err = net.Dial("tcp", "127.0.0.1:2046")
			Verify(c, NotNil)
			Verify(err, Equal, nil)
			Verify(tcpReqCount, Equal, 2)

			tcpBytes = []byte("h")
			_, err = c.Write(tcpBytes)
			Verify(err, Equal, nil)
			time.Sleep(time.Second)
			Verify(tcpReadBytes, Equal, len(tcpBytes))
		})

		TearDown(func() {
			m.Stop()
			_ = h1Server.Shutdown(context.TODO())
			_ = tcpListener.Close()
			monkey.UnpatchAll()
		})
	})
}

const ConfigSimpleHTTP1 = `{
  "servers": [
    {
      "default_log_path": "stdout",
      "default_log_level": "ERROR",
      "routers": [
        {
          "router_config_name": "router_to_server",
          "virtual_hosts": [
            {
              "name": "server_hosts",
              "domains": [
                "*"
              ],
              "routers": [
                {
                  "match": {
                    "prefix": "/"
                  },
                  "route": {
                    "cluster_name": "server_http_cluster"
                  }
                }
              ]
            }
          ]
        }
      ],
      "listeners": [
        {
          "address": "127.0.0.1:2046",
          "bind_port": true,
          "filter_chains": [
            {
              "filters": [
                {
                  "type": "proxy",
                  "config": {
					"fallback_for_unknown_protocol": true,
                    "downstream_protocol": "Auto",
                    "upstream_protocol": "Http1",
                    "router_config_name": "router_to_server"
                  }
                },
                {
                  "type": "tcp_proxy",
                  "config": {
                    "cluster": "server_tcp_cluster"
                  }
                }
              ]
            }
          ]
        }
      ]
    }
  ],
  "cluster_manager": {
    "clusters": [
      {
        "name": "server_http_cluster",
        "type": "SIMPLE",
        "lb_type": "LB_RANDOM",
        "hosts": [
          {
            "address": "127.0.0.1:9091"
          }
        ]
      },
      {
        "name": "server_tcp_cluster",
        "type": "SIMPLE",
        "lb_type": "LB_RANDOM",
        "hosts": [
          {
            "address": "127.0.0.1:9092"
          }
        ]
      }
    ]
  }
}`
