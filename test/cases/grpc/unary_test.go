//go:build MOSNTest
// +build MOSNTest

package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
)

func TestSimpleGrpc(t *testing.T) {
	Scenario(t, "simple grpc server in mosn networkfilter", func() {
		_, _ = lib.InitMosn(ConfigHelloGrpcFilter) // no servers need
		Case("call grpc", func() {
			conn, err := grpc.Dial("127.0.0.1:2045", grpc.WithInsecure(), grpc.WithBlock())
			Verify(err, Equal, nil)
			defer conn.Close()
			c := pb.NewGreeterClient(conn)
			r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: "mosn grpc"})
			Verify(err, Equal, nil)
			Verify(r.GetMessage(), Equal, "Hello mosn grpc")
		})
	})
}

const ConfigHelloGrpcFilter = `{
	"servers":[
		{
			"default_log_path":"stdout",
			"default_log_level":"DEBUG",
			"listeners":[
				{
					"address":"127.0.0.1:2045",
					"bind_port": true,
					"filter_chains": [{
						"filters": [
							{
								"type":"grpc",
								"config": {
									"server_name":"hello"
								}
							}
						]
					}]
				}
			]
		}
	]
}`
