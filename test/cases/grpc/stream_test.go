//go:build MOSNTest
// +build MOSNTest

package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/features/proto/echo"
	"google.golang.org/grpc/metadata"
	"mosn.io/mosn/pkg/log"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib"
)

const (
	timestampFormat = time.StampNano // "Jan _2 15:04:05.000"
	streamingCount  = 10
)

func TestStreamGrpc(t *testing.T) {
	Scenario(t, "streaming grpc server is mosn networkfilter", func() {
		_, _ = lib.InitMosn(ConfigStreamGrpcFilter) // no servers need
		Case("server streaming", func() {
			conn, err := grpc.Dial("127.0.0.1:2045", grpc.WithInsecure(), grpc.WithBlock())
			Verify(err, Equal, nil)
			defer conn.Close()
			c := pb.NewEchoClient(conn)
			message := "this is server streaming metadata"
			Verify(serverStreamingWithMetadata(c, message), Equal, nil)
		})
		Case("client streaming", func() {
			conn, err := grpc.Dial("127.0.0.1:2045", grpc.WithInsecure(), grpc.WithBlock())
			Verify(err, Equal, nil)
			defer conn.Close()
			c := pb.NewEchoClient(conn)
			message := "this is client streaming metadata"
			Verify(clientStreamWithMetadata(c, message), Equal, nil)
		})
		Case("bidrectional", func() {
			conn, err := grpc.Dial("127.0.0.1:2045", grpc.WithInsecure(), grpc.WithBlock())
			Verify(err, Equal, nil)
			defer conn.Close()
			c := pb.NewEchoClient(conn)
			message := "this is bidrectional metadata"
			Verify(bidirectionalWithMetadata(c, message), Equal, nil)
		})
	})
}

// stream grpc client is implemented with reference to grpc/examples/features/proto/echo
func serverStreamingWithMetadata(c pb.EchoClient, message string) error {
	// Create metadata and context.
	md := metadata.Pairs("timestamp", time.Now().Format(timestampFormat))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	// Make RPC using the context with the metadata.
	stream, err := c.ServerStreamingEcho(ctx, &pb.EchoRequest{Message: message})
	if err != nil {
		log.DefaultLogger.Errorf("failed to call ServerStreamingEcho: %v", err)
		return err
	}
	// Read the header when the header arrives.
	header, err := stream.Header()
	if err != nil {
		log.DefaultLogger.Errorf("failed to get header from stream: %v", err)
		return err
	}
	//  Read metadata from server's header.
	if _, ok := header["timestamp"]; !ok {
		log.DefaultLogger.Errorf("timestamp expected but doesn't exist in header")
		return errors.New("timestamp expected but doesn't exist in header")
	}
	if _, ok := header["location"]; !ok {
		log.DefaultLogger.Errorf("location expected but doesn't exist in header")
		return errors.New("location expected but doesn't exist in header")
	}
	// Read all the responses.
	var rpcStatus error
	respCount := 0
	for {
		r, err := stream.Recv()
		if err != nil {
			rpcStatus = err
			break
		}
		if r.Message != message {
			rpcStatus = fmt.Errorf("unexpected response message: %s", r.Message)
			break
		}
		respCount++
	}
	if rpcStatus != io.EOF || respCount != streamingCount {
		log.DefaultLogger.Errorf("failed to finish server streaming: %v, response count: %d", rpcStatus, respCount)
		return errors.New("unexpected response")
	}
	// Read the trailer after the RPC is finished.
	trailer := stream.Trailer()
	if _, ok := trailer["timestamp"]; !ok {
		log.DefaultLogger.Errorf("timestamp expected but doesn't exist in trailer")
		return errors.New("timestamp expected but doesn't exist in trailer")
	}
	return nil
}

func clientStreamWithMetadata(c pb.EchoClient, message string) error {
	// Create metadata and context.
	md := metadata.Pairs("timestamp", time.Now().Format(timestampFormat))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	// Make RPC using the context with the metadata.
	stream, err := c.ClientStreamingEcho(ctx)
	if err != nil {
		log.DefaultLogger.Errorf("failed to get header from stream: %v", err)
		return err
	}
	// Read the header when the header arrives.
	header, err := stream.Header()
	if err != nil {
		log.DefaultLogger.Errorf("failed to get header from stream: %v", err)
		return err
	}
	//  Read metadata from server's header.
	if _, ok := header["timestamp"]; !ok {
		log.DefaultLogger.Errorf("timestamp expected but doesn't exist in header")
		return errors.New("timestamp expected but doesn't exist in header")
	}
	if _, ok := header["location"]; !ok {
		log.DefaultLogger.Errorf("location expected but doesn't exist in header")
		return errors.New("location expected but doesn't exist in header")
	}
	// Send all requests to the server.
	for i := 0; i < streamingCount; i++ {
		if err := stream.Send(&pb.EchoRequest{Message: message}); err != nil {
			log.DefaultLogger.Errorf("failed to send streaming: %v", err)
			return err
		}
	}
	// Read the response.
	r, err := stream.CloseAndRecv()
	if err != nil {
		log.DefaultLogger.Errorf("failed to CloseAndRecv: %v", err)
		return err
	}
	if r.Message != message {
		return fmt.Errorf("unexpected response message: %s", r.Message)
	}
	// Read the trailer after the RPC is finished.
	trailer := stream.Trailer()
	if _, ok := trailer["timestamp"]; !ok {
		log.DefaultLogger.Errorf("timestamp expected but doesn't exist in trailer")
		return errors.New("timestamp expected but doesn't exist in trailer")
	}
	return nil
}

func bidirectionalWithMetadata(c pb.EchoClient, message string) error {
	// Create metadata and context.
	md := metadata.Pairs("timestamp", time.Now().Format(timestampFormat))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	// Make RPC using the context with the metadata.
	stream, err := c.BidirectionalStreamingEcho(ctx)
	if err != nil {
		log.DefaultLogger.Errorf("failed to call BidirectionalStreamingEcho: %v", err)
		return err
	}
	errCh := make(chan error)
	go func() {
		// Read the header when the header arrives.
		header, err := stream.Header()
		if err != nil {
			log.DefaultLogger.Errorf("failed to get header from stream: %v", err)
			errCh <- err
			return
		}
		//  Read metadata from server's header.
		if _, ok := header["timestamp"]; !ok {
			log.DefaultLogger.Errorf("timestamp expected but doesn't exist in header")
			errCh <- errors.New("timestamp expected but doesn't exist in header")
			return
		}
		if _, ok := header["location"]; !ok {
			log.DefaultLogger.Errorf("location expected but doesn't exist in header")
			errCh <- errors.New("location expected but doesn't exist in header")
			return
		}
		// Send all requests to the server.
		for i := 0; i < streamingCount; i++ {
			if err := stream.Send(&pb.EchoRequest{Message: message}); err != nil {
				log.DefaultLogger.Errorf("failed to send streaming: %v", err)
				errCh <- err
				return
			}
		}
		stream.CloseSend()
	}()
	// Read all the responses.
	var rpcStatus error
	respCount := 0
RESP:
	for {
		select {
		case err := <-errCh:
			rpcStatus = err
			break RESP
		default:
			r, err := stream.Recv()
			if err != nil {
				rpcStatus = err
				break RESP
			}
			if r.Message != message {
				rpcStatus = fmt.Errorf("unexpected response message: %s", r.Message)
				break
			}
			respCount++
		}
	}
	if rpcStatus != io.EOF || respCount != streamingCount {
		log.DefaultLogger.Errorf("failed to finish server streaming: %v, response count: %d", rpcStatus, respCount)
		return errors.New("unexpected response")
	}
	// Read the trailer after the RPC is finished.
	trailer := stream.Trailer()
	if _, ok := trailer["timestamp"]; !ok {
		log.DefaultLogger.Errorf("timestamp expected but doesn't exist in trailer")
		return errors.New("timestamp expected but doesn't exist in trailer")
	}
	return nil
}

const ConfigStreamGrpcFilter = `{
	"servers":[
		{
			"default_log_path":"stdout",
			"default_log_level":"ERROR",
			"listeners":[
				{
					"address":"127.0.0.1:2045",
					"bind_port": true,
					"filter_chains": [{
						"filters": [
							{
								"type":"grpc",
								"config": {
									"server_name":"echo"
								}
							}
						]
					}]
				}
			]
		}
	]
}`
