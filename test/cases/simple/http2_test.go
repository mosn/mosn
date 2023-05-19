//go:build MOSNTest
// +build MOSNTest

package simple

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net"
	goHttp "net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	pb "google.golang.org/grpc/examples/features/proto/echo"
	"google.golang.org/grpc/status"

	"mosn.io/mosn/pkg/module/http2/h2c"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
)

var largeBody = make([]byte, 1<<18)

func TestHttp2NotUseStream(t *testing.T) {
	Scenario(t, "http2 not use stream", func() {
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
					reqBody: []byte("xxxxx"),
				},
				{
					reqBody: largeBody,
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
				resp, err := client.Post("http://localhost:2046", "text/plain", bytes.NewReader(tc.reqBody))
				Verify(err, Equal, nil)
				respBody, err := ioutil.ReadAll(resp.Body)
				Verify(err, Equal, nil)
				Verify(len(respBody), Equal, len(tc.reqBody))
			}
		})
		TearDown(func() {
			m.Stop()
			_ = server.Shutdown(context.TODO())
		})
	})
}

func TestHttp2UseStream(t *testing.T) {
	Scenario(t, "http2 use stream", func() {
		var m *mosn.MosnOperator
		var server *goHttp.Server
		Setup(func() {
			m = mosn.StartMosn(ConfigSimpleHTTP2UseStream)
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
					reqBody: []byte("xxxxx"),
				},
				{
					reqBody: largeBody,
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
				resp, err := client.Post("http://localhost:2046", "text/plain", bytes.NewReader(tc.reqBody))
				Verify(err, Equal, nil)
				respBody, err := ioutil.ReadAll(resp.Body)
				Verify(err, Equal, nil)
				Verify(len(respBody), Equal, len(tc.reqBody))
			}
		})
		TearDown(func() {
			m.Stop()
			_ = server.Shutdown(context.TODO())
		})
	})
}

type bidirectionalStreamingGrpcServer struct {
	pb.UnimplementedEchoServer
}

func (s *bidirectionalStreamingGrpcServer) BidirectionalStreamingEcho(stream pb.Echo_BidirectionalStreamingEchoServer) error {
	for {
		in, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		stream.Send(&pb.EchoResponse{Message: in.Message})
	}
}

func TestGRPCBidirectionalStreaming(t *testing.T) {
	Scenario(t, "http2 use stream", func() {
		var m *mosn.MosnOperator
		var server *grpc.Server
		Setup(func() {
			m = mosn.StartMosn(ConfigSimpleHTTP2UseStream)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("client-mosn-server", func() {
			go func() {
				lis, err := net.Listen("tcp", "0.0.0.0:8080")
				Verify(err, Equal, nil)
				server = grpc.NewServer()
				pb.RegisterEchoServer(server, &bidirectionalStreamingGrpcServer{})
				server.Serve(lis)
			}()

			time.Sleep(time.Second)

			// Set up a connection to the server.
			conn, err := grpc.Dial("127.0.0.1:2046", grpc.WithInsecure())
			Verify(err, Equal, nil)
			defer conn.Close()

			c := pb.NewEchoClient(conn)

			testcases := []struct {
				reqMessage string
			}{
				{
					reqMessage: "hello world",
				},
			}

			for _, testcase := range testcases {
				// Initiate the stream with a context that supports cancellation.
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				stream, err := c.BidirectionalStreamingEcho(ctx)
				Verify(err, Equal, nil)
				// Send some test messages.
				for i := 0; i < 5; i++ {
					err = stream.Send(&pb.EchoRequest{Message: testcase.reqMessage})
					Verify(err, Equal, nil)

					res, err := stream.Recv()
					Verify(status.Code(err), Equal, codes.OK)
					Verify(res.Message, Equal, testcase.reqMessage)
				}
				cancel()
			}
		})
		TearDown(func() {
			m.Stop()
			server.GracefulStop()
		})
	})
}

func TestHttp2UseStreamNoRetry(t *testing.T) {
	Scenario(t, "http2 use stream no retry", func() {
		var m *mosn.MosnOperator
		var serverHTTP2, serverHTTP1 *goHttp.Server
		Setup(func() {
			m = mosn.StartMosn(ConfigSimpleHTTP2UseStreamNoRetry)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("client-mosn-server", func() {
			go func() {
				h2s := &http2.Server{}
				handler := goHttp.HandlerFunc(func(w goHttp.ResponseWriter, r *goHttp.Request) {
					_, err := ioutil.ReadAll(r.Body)
					if err != nil {
						t.Fatalf(err.Error())
					}
					w.WriteHeader(200)
				})
				serverHTTP2 = &goHttp.Server{
					Addr:    "0.0.0.0:8080",
					Handler: h2c.NewHandler(handler, h2s),
				}
				_ = serverHTTP2.ListenAndServe()
			}()
			// http1 protocol error
			// https://github.com/mosn/mosn/issues/1751
			go func() {
				serverHTTP1 = &goHttp.Server{Addr: "0.0.0.0:8081", Handler: nil}
				serverHTTP1.ListenAndServe()
			}()

			time.Sleep(time.Second)

			client := goHttp.Client{
				Timeout: time.Second * 2,
				Transport: &http2.Transport{
					AllowHTTP: true,
					DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
						return net.Dial(network, addr)
					},
				},
			}
			for i := 0; i < 10; i++ {
				_, err := client.Post("http://localhost:2046", "text/plain", bytes.NewReader(largeBody))
				if err != nil && strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
					t.Fatalf("error request hangs")
				}
			}
		})
		TearDown(func() {
			m.Stop()
			_ = serverHTTP1.Shutdown(context.TODO())
			_ = serverHTTP2.Shutdown(context.TODO())
		})
	})
}

type unaryGrpcServer struct {
	pb.UnimplementedEchoServer
	event chan struct{}
}

func (s *unaryGrpcServer) UnaryEcho(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	select {
	case <-ctx.Done():
		close(s.event)
	case <-time.After(time.Second):
	}
	return &pb.EchoResponse{}, nil
}

func TestHttp2TransferResetStream(t *testing.T) {
	Scenario(t, "http2 transfer reset stream", func() {
		var m *mosn.MosnOperator
		var server *grpc.Server
		Setup(func() {
			m = mosn.StartMosn(ConfigSimpleHTTP2UseStream)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("client-mosn-server", func() {
			serverEvent := make(chan struct{})
			go func() {
				lis, err := net.Listen("tcp", "0.0.0.0:8080")
				Verify(err, Equal, nil)
				server = grpc.NewServer()
				pb.RegisterEchoServer(server, &unaryGrpcServer{
					event: serverEvent,
				})
				server.Serve(lis)
			}()

			time.Sleep(time.Second)

			// Set up a connection to the server.
			conn, err := grpc.Dial("127.0.0.1:2046", grpc.WithInsecure())
			Verify(err, Equal, nil)
			defer conn.Close()

			c := pb.NewEchoClient(conn)

			// Initiate the stream with a context that supports cancellation.
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(500 * time.Millisecond)
				cancel()
			}()
			_, err = c.UnaryEcho(ctx, &pb.EchoRequest{
				Message: "hello jack",
			})
			Verify(err, NotEqual, nil)
			select {
			case <-serverEvent:
				t.Log("context transferred successfully")
			case <-time.After(500 * time.Millisecond):
				t.Fatalf("error context not transferred")
			}
		})
		TearDown(func() {
			m.Stop()
			server.GracefulStop()
		})
	})
}

const ConfigSimpleHTTP2UseStream = `{
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
                                                                        "router_config_name":"router_to_server",
                                                                        "extend_config": {
                                                                                "http2_use_stream": true
                                                                        }
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

const ConfigSimpleHTTP2UseStreamNoRetry = `{
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
                                                                        "router_config_name":"router_to_server",
                                                                        "extend_config": {
                                                                                "http2_use_stream": true
                                                                        }
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
                                        {"address":"127.0.0.1:8080"},
                                        {"address":"127.0.0.1:8081"}
                                ]
                        }
                ]
        }
}`
