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

package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/cmd/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

//when a upstream server has been closed
//the client should get a error response
func TestServerClose(t *testing.T) {
	meshAddr := "127.0.0.1:2045"
	serverAddrs := []string{
		"127.0.0.1:8080",
		"127.0.0.1:8081",
	}
	servers := []*UpstreamServer{}
	for _, addr := range serverAddrs {
		server := NewUpstreamServer(t, addr, ServeBoltV1)
		server.GoServe()
		defer server.Close()
		servers = append(servers, server)
	}
	meshConfig := CreateSimpleMeshConfig(meshAddr, serverAddrs, protocol.SofaRPC, protocol.SofaRPC)
	mesh := mosn.NewMosn(meshConfig)
	go mesh.Start()
	defer mesh.Close()
	time.Sleep(5 * time.Second) //wait mesh and server start
	client := &BoltV1Client{
		t:        t,
		ClientID: "testClient",
		Waits:    sync.Map{},
	}
	client.Connect(meshAddr)
	defer client.conn.Close(types.NoFlush, types.LocalClose)
	//send request
	go func() {
		for i := 0; i < 10; i++ {
			client.SendRequest()
			time.Sleep(time.Second)
		}
	}()
	//close a server after 4 seconds
	go func() {
		<-time.After(4 * time.Second)
		servers[0].Close()
	}()
	<-time.After(15 * time.Second) //wait request finish
	if !IsMapEmpty(&client.Waits) {
		t.Errorf("some request get no response\n")
	}
}
