package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	echopb "google.golang.org/grpc/examples/features/proto/echo"
	hellopb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	mgrpc "mosn.io/mosn/pkg/filter/network/grpc"
)

func init() {
	mgrpc.RegisterServerHandler("hello", NewHelloExampleGrpcServer)
	mgrpc.RegisterServerHandler("echo", NewEchoStreamGrpcServer)
}

// helloServer is used to implement helloworld.GreeterServer.
type helloServer struct {
	hellopb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *helloServer) SayHello(ctx context.Context, in *hellopb.HelloRequest) (*hellopb.HelloReply, error) {
	return &hellopb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func NewHelloExampleGrpcServer(_ json.RawMessage, options ...grpc.ServerOption) (mgrpc.RegisteredServer, error) {
	s := grpc.NewServer(options...)
	hellopb.RegisterGreeterServer(s, &helloServer{})
	return s, nil
}

const (
	timestampFormat = time.StampNano // "Jan _2 15:04:05.000"
	streamingCount  = 10
)

// echoServer is used to implement echo.EchoServer
type echoServer struct {
	echopb.UnimplementedEchoServer
}

func (s *echoServer) UnaryEcho(ctx context.Context, in *echopb.EchoRequest) (*echopb.EchoResponse, error) {
	fmt.Printf("--- UnaryEcho ---\n")
	// Create trailer in defer to record function return time.
	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(timestampFormat))
		grpc.SetTrailer(ctx, trailer)
	}()

	// Read metadata from client.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.DataLoss, "UnaryEcho: failed to get metadata")
	}
	if t, ok := md["timestamp"]; ok {
		fmt.Printf("timestamp from metadata:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	}

	// Create and send header.
	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(timestampFormat)})
	grpc.SendHeader(ctx, header)

	fmt.Printf("request received: %v, sending echo\n", in)

	return &echopb.EchoResponse{Message: in.Message}, nil
}

func (s *echoServer) ServerStreamingEcho(in *echopb.EchoRequest, stream echopb.Echo_ServerStreamingEchoServer) error {
	fmt.Printf("--- ServerStreamingEcho ---\n")
	// Create trailer in defer to record function return time.
	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(timestampFormat))
		stream.SetTrailer(trailer)
	}()

	// Read metadata from client.
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "ServerStreamingEcho: failed to get metadata")
	}
	if t, ok := md["timestamp"]; ok {
		fmt.Printf("timestamp from metadata:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	}

	// Create and send header.
	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(timestampFormat)})
	stream.SendHeader(header)

	fmt.Printf("request received: %v\n", in)

	// Read requests and send responses.
	for i := 0; i < streamingCount; i++ {
		fmt.Printf("echo message %v\n", in.Message)
		err := stream.Send(&echopb.EchoResponse{Message: in.Message})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *echoServer) ClientStreamingEcho(stream echopb.Echo_ClientStreamingEchoServer) error {
	fmt.Printf("--- ClientStreamingEcho ---\n")
	// Create trailer in defer to record function return time.
	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(timestampFormat))
		stream.SetTrailer(trailer)
	}()

	// Read metadata from client.
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "ClientStreamingEcho: failed to get metadata")
	}
	if t, ok := md["timestamp"]; ok {
		fmt.Printf("timestamp from metadata:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	}

	// Create and send header.
	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(timestampFormat)})
	stream.SendHeader(header)

	// Read requests and send responses.
	var message string
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			fmt.Printf("echo last received message\n")
			return stream.SendAndClose(&echopb.EchoResponse{Message: message})
		}
		message = in.Message
		fmt.Printf("request received: %v, building echo\n", in)
		if err != nil {
			return err
		}
	}
}

func (s *echoServer) BidirectionalStreamingEcho(stream echopb.Echo_BidirectionalStreamingEchoServer) error {
	fmt.Printf("--- BidirectionalStreamingEcho ---\n")
	// Create trailer in defer to record function return time.
	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(timestampFormat))
		stream.SetTrailer(trailer)
	}()

	// Read metadata from client.
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "BidirectionalStreamingEcho: failed to get metadata")
	}

	if t, ok := md["timestamp"]; ok {
		fmt.Printf("timestamp from metadata:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	}

	// Create and send header.
	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(timestampFormat)})
	stream.SendHeader(header)

	// Read requests and send responses.
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Printf("request received %v, sending echo\n", in)
		if err := stream.Send(&echopb.EchoResponse{Message: in.Message}); err != nil {
			return err
		}
	}
}

func NewEchoStreamGrpcServer(_ json.RawMessage, options ...grpc.ServerOption) (mgrpc.RegisteredServer, error) {
	rand.Seed(time.Now().UnixNano())
	s := grpc.NewServer(options...)
	echopb.RegisterEchoServer(s, &echoServer{})
	return s, nil
}
