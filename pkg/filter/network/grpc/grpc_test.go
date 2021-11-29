/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package grpc

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/pkg/buffer"
)

func TestGrpcFilter(t *testing.T) {
	RegisterServerHandler("test", NewHelloExampleGrpcServer)
	addr := "127.0.0.1:8080"
	param := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       "test",
			AddrConfig: addr,
		},
	}
	streamfilter.GetStreamFilterManager().AddOrUpdateStreamFilterConfig(param.Name, param.StreamFilters)
	factory, err := CreateGRPCServerFilterFactory(map[string]interface{}{
		"server_name": "test",
	})
	if err != nil {
		t.Fatalf("create filter failed: %v", err)
	}
	if initialzer, ok := factory.(api.FactoryInitializer); ok {
		if err := initialzer.Init(param); err != nil {
			t.Fatalf("initialize factory failed: %v", err)
		}
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()
	// wait server start
	time.Sleep(2 * time.Second)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		// mock receive a listener
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// mock create a new connection
		cb := &MockFactoryCallbacks{}
		factory.CreateFilterChain(context.Background(), cb)
		filter := cb.rf
		rfcb := &MockReadFilterCallbacks{
			conn: &MockConnection{
				raw: conn,
			},
		}
		defer func() {
			gf, ok := filter.(*grpcFilter)
			if ok {
				gf.conn.OnEvent(api.RemoteClose)
				gf.ln.Close()
			}
		}()
		filter.InitializeReadFilterCallbacks(rfcb)
		filter.OnNewConnection()
		for {
			b := make([]byte, 512)
			conn.SetDeadline(time.Now().Add(5 * time.Second))
			n, err := conn.Read(b)
			if err != nil {
				return
			}
			buf := buffer.NewIoBuffer(0)
			buf.Write(b[:n])
			filter.OnData(buf)
		}
	}()
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("grpc Dial returns an error: %v", err)
	}
	defer conn.Close()

	c := pb.NewGreeterClient(conn)

	// mock grpc client
	testHello(c, t)

	wg.Wait()
}

func testHello(c pb.GreeterClient, t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: "test"})
	if err != nil {
		t.Fatalf("grpc client returns an error: %v", err)
	}
	if r.Message != "Hello test" {
		t.Fatalf("unexpected result: %s", r.Message)
	}
}

// Mock a gRPC Server for test
// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func NewHelloExampleGrpcServer(_ json.RawMessage, options ...grpc.ServerOption) (RegisteredServer, error) {
	s := grpc.NewServer(options...)
	pb.RegisterGreeterServer(s, &server{})
	return s, nil
}

// Mock a filter chain callbacks and read filter callbacks
type MockFactoryCallbacks struct {
	api.NetWorkFilterChainFactoryCallbacks
	rf api.ReadFilter
}

func (cb *MockFactoryCallbacks) AddReadFilter(rf api.ReadFilter) {
	cb.rf = rf
}

type MockReadFilterCallbacks struct {
	api.ReadFilterCallbacks
	conn *MockConnection
}

func (cb *MockReadFilterCallbacks) Connection() api.Connection {
	return cb.conn
}

type MockConnection struct {
	api.Connection
	raw    net.Conn
	closed bool
}

func (c *MockConnection) Write(buf ...buffer.IoBuffer) error {
	for _, b := range buf {
		_, err := c.raw.Write(b.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *MockConnection) LocalAddr() net.Addr {
	if c.raw == nil {
		return nil
	}
	return c.raw.LocalAddr()
}

func (c *MockConnection) RemoteAddr() net.Addr {
	if c.raw == nil {
		return nil
	}
	return c.raw.RemoteAddr()
}

func (c *MockConnection) RawConn() net.Conn {
	return c.raw
}

func (c *MockConnection) AddConnectionEventListener(api.ConnectionEventListener) {
}

func (c *MockConnection) Close(_ api.ConnectionCloseType, _ api.ConnectionEvent) error {
	c.closed = true
	if c.raw == nil {
		return nil
	}
	return c.raw.Close()
}
