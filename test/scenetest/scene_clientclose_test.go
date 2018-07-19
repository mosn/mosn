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
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/cmd/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

//one client close should not effect others
func TestClientClose(t *testing.T) {
	sofaAddr := "127.0.0.1:8080"
	meshAddr := "127.0.0.1:2045"
	server := NewUpstreamServer(t, sofaAddr, ServeBoltV1)
	server.GoServe()
	defer server.Close()
	meshConfig := CreateSimpleMeshConfig(meshAddr, []string{sofaAddr}, protocol.SofaRPC, protocol.SofaRPC)
	mesh := mosn.NewMosn(meshConfig)
	go mesh.Start()
	defer mesh.Close()
	time.Sleep(5 * time.Second) //wait mesh and server start
	var stopChans []chan struct{}
	//send request with cancel
	wg := sync.WaitGroup{}

	call := func(client *BoltV1Client, stop chan struct{}) {
		defer wg.Done()
		for {
			select {
			case <-stop:
				//before close, should check all request get response
				<-time.After(time.Second)
				if !IsMapEmpty(&client.Waits) {
					t.Errorf("client %s has request timeout\n", client.ClientID)
				}
				client.conn.Close(types.NoFlush, types.LocalClose)
				client.Stats()
				return
			default:
				client.SendRequest()
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	makeClient := func(clientId string, stop chan struct{}) {
		client := &BoltV1Client{
			t:        t,
			ClientID: clientId,
			Waits:    sync.Map{},
		}
		if err := client.Connect(meshAddr); err != nil {
			t.Fatalf("client %s connect to mesh failed, error: %v\n", clientId, err)
		}
		wg.Add(1)
		go call(client, stop)
	}
	//create 2 client
	for i := 0; i < 2; i++ {
		stop := make(chan struct{})
		stopChans = append(stopChans, stop)
		makeClient(fmt.Sprintf("client.%d", i), stop)
	}
	//create a client will be closed
	stop := make(chan struct{})
	makeClient("client.close", stop)
	//close the client randomly, 1s ~ 3s
	closetime := time.Duration(rand.Intn(2000)) * time.Millisecond
	<-time.After(time.Second + closetime)
	close(stop)
	//create a new client
	<-time.After(3 * time.Second)
	ch := make(chan struct{})
	stopChans = append(stopChans, ch)
	makeClient("client.new", ch)
	//close all client
	<-time.After(5 * time.Second)
	for _, cancel := range stopChans {
		close(cancel)
	}
	//wait close and verify with timeout
	ok := make(chan struct{})
	go func() {
		wg.Wait()
		close(ok)
	}()
	select {
	case <-time.After(10 * time.Second):
		t.Errorf("test timeout\n")
	case <-ok:
	}
}
