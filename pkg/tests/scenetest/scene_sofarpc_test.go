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
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/orcaman/concurrent-map"
)

func TestSofaRpc(t *testing.T) {
	sofaAddr := "127.0.0.1:8080"
	meshAddr := "127.0.0.1:2045"
	server := NewUpstreamServer(t, sofaAddr, ServeBoltV1)
	server.GoServe()
	defer server.Close()
	meshConfig := CreateSimpleMeshConfig(meshAddr, []string{sofaAddr}, protocol.SofaRpc, protocol.SofaRpc)
	mesh := mosn.NewMosn(meshConfig)
	go mesh.Start()
	defer mesh.Close()
	time.Sleep(5 * time.Second) //wait mesh and server start
	//client
	client := &BoltV1Client{
		t:        t,
		ClientId: "testClient",
		Waits:    cmap.New(),
	}
	client.Connect(meshAddr)
	defer client.conn.Close(types.NoFlush, types.LocalClose)
	for i := 0; i < 20; i++ {
		client.SendRequest()
	}
	<-time.After(10 * time.Second)
	if !client.Waits.IsEmpty() {
		t.Errorf("exists request no response\n")
	}
}
