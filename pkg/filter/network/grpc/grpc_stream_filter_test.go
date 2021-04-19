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
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/streamfilter"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	_ "mosn.io/mosn/pkg/filter/stream/flowcontrol"
	"mosn.io/pkg/buffer"
)

func TestFlowControlFilter(t *testing.T) {
	RegisterServerHandler("test", NewHelloExampleGrpcServer)
	addr := "127.0.0.1:8080"

	streamFilterConfig := []byte(`[
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
														"threshold": 5,
														"strategy": 0
													}
												]
											}
										}
									]`)

	filter := new([]v2.Filter)
	err := json.Unmarshal(streamFilterConfig, filter)
	if err != nil {
		t.Fatal(err)
	}
	param := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name: "test",
			AddrConfig: addr,
			StreamFilters: *filter,
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
			conn.SetDeadline(time.Now().Add(2 * time.Second))
			n, err := conn.Read(b)
			if err != nil {
				return
			}
			buf := buffer.NewIoBuffer(0)
			buf.Write(b[:n])
			filter.OnData(buf)
		}
	}()
	// mock grpc client
	multipleCall(addr, t)
	wg.Wait()
}

func multipleCall(addr string, t *testing.T) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		r, err := c.SayHello(ctx, &pb.HelloRequest{Name: "test"})
		if i <= 4 {
			assert.Equal(t, r.Message, "Hello test")
		}else{
			assert.NotNil(t, err.Error(), "current request is limited")
		}
		cancel()
	}
}
