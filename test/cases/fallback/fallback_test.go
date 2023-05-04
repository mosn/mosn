//go:build MOSNTest
// +build MOSNTest

package fallback

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
)

var receiveHttpReq bool = false
var receiveTcpReq bool = true

func startTcpServer(port int) net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil || l == nil {
		fmt.Println("fail to start tcp server")
		return nil
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				fmt.Println("fail to accept tcp connection")
				return
			}
			go func(conn net.Conn) {
				fmt.Println("Accepted new connection.")
				defer conn.Close()

				for {
					buf := make([]byte, 1024)
					size, err := conn.Read(buf)
					if err != nil {
						return
					}
					data := buf[:size]
					fmt.Println("Read new data from connection:", string(data))
					conn.Write(data)
					receiveTcpReq = true
				}
			}(conn)
		}
	}()
	return l
}

func startHttpServer(port int) *http.Server {
	server := &http.Server{Addr: "127.0.0.1:" + strconv.Itoa(port)}
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		receiveHttpReq = true
	})
	go server.ListenAndServe()
	return server
}

func TestProxyFallback(t *testing.T) {
	Scenario(t, "proxy fallback", func() {
		var m *mosn.MosnOperator
		var tcpServer net.Listener
		var httpServer *http.Server

		Setup(func() {
			// start mosn
			m = mosn.StartMosn(ConfigFallback)
			Verify(m, NotNil)
			// start http server
			httpServer = startHttpServer(22164)
			// start tcp server
			tcpServer = startTcpServer(22165)
			// wait servers start
			time.Sleep(2 * time.Second)
		})

		Case("client-mosn-server", func() {
			// http req
			_, _ = http.Get("http://127.0.0.1:22163/")

			// tcp req
			c, err := net.Dial("tcp", "127.0.0.1:22163")
			Verify(err, Equal, nil)
			c.Write([]byte("hello world hello world\n"))

			time.Sleep(time.Second)

			// read tcp resp
			buf := make([]byte, 1024)
			size, err := c.Read(buf)
			Verify(err, Equal, nil)
			data := buf[:size]

			Verify(receiveHttpReq, Equal, true)
			Verify(receiveTcpReq, Equal, true)
			Verify(string(data), Equal, "hello world hello world\n")

			// no data, wait until time out
			_, err = net.Dial("tcp", "127.0.0.1:22163")
			Verify(err, Equal, nil)
			time.Sleep(18 * time.Second)
		})

		TearDown(func() {
			m.Stop()
			_ = tcpServer.Close()
			_ = httpServer.Close()
		})
	})
}

const ConfigFallback = `
{
  "servers": [
    {
      "default_log_path": "stdout",
      "routers": [
        {
          "router_config_name": "router_to_http_server",
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
          "address": "127.0.0.1:22163",
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
                    "router_config_name": "router_to_http_server"
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
            "address": "127.0.0.1:22164"
          }
        ]
      },
      {
        "name": "server_tcp_cluster",
        "type": "SIMPLE",
        "lb_type": "LB_RANDOM",
        "hosts": [
          {
            "address": "127.0.0.1:22165"
          }
        ]
      }
    ]
  }
}
`
