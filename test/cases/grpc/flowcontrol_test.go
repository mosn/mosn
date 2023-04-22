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

func TestGrpcFlowControl(t *testing.T) {
	Scenario(t, "flow control grpc server in mosn", func() {
		_, _ = lib.InitMosn(ConfigFlowControlGrpcFilter) // no servers need
		Case("call grpc", func() {
			conn, err := grpc.Dial("127.0.0.1:2045", grpc.WithInsecure(), grpc.WithBlock())
			Verify(err, Equal, nil)
			defer conn.Close()
			c := pb.NewGreeterClient(conn)
			for i := 0; i < 2; i++ {
				r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: "mosn grpc"})
				if i == 0 {
					Verify(err, Equal, nil)
					Verify(r.GetMessage(), Equal, "Hello mosn grpc")
				} else if i == 1 {
					Verify(err, NotNil)
					Verify(err.Error(), Equal, "rpc error: code = Unknown desc = current request is limited")
				}
			}
		})
	})
}

const ConfigFlowControlGrpcFilter = `{
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
					}],
					"stream_filters": [
						{
							"type": "flowControlFilter",
							"config": {
								"global_switch": true,
								"monitor": false,
								"limit_key_type": "PATH",
								"rules": [
									{
										"resource": "/helloworld.Greeter/SayHello",
										"limitApp": "",
										"grade": 1,
										"threshold": 1,
										"strategy": 0
									}
								]
							}
						}
					]
				}
			]
		}
	]
}`
