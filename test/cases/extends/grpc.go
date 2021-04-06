package main

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	mgrpc "mosn.io/mosn/pkg/filter/network/grpc"
)

func init() {
	mgrpc.RegisterServerHandler("hello", NewHelloExampleGrpcServer)
}

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func NewHelloExampleGrpcServer(_ json.RawMessage) *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	return s
}
