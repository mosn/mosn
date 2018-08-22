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
 
package integrate

import (
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	testutil "github.com/alipay/sofa-mosn/test/util"
)

type weightCase struct {
	*testCase
	appServers  []testutil.UpstreamServer
}

var hostsAddress = []string{"127.0.0.1:8081","127.0.0.1:8082","127.0.0.1:8083","127.0.0.1:8084"}
func (c *weightCase) Start() {
	for _,appserver := range c.appServers {
		appserver.GoServe()
	}
	
	meshAddr := testutil.CurrentMeshAddr()
	c.clientMeshAddr = meshAddr
	cfg := testutil.CreateWeightProxyMesh(meshAddr, hostsAddress, protocol.SofaRPC)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.stop
		for _,appserver := range c.appServers {
			appserver.Close()
		}
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func TestWeightProxy(t *testing.T) {
	testCases := []*weightCase{
		{
			testCase: newTestCase(t, protocol.SofaRPC, protocol.SofaRPC, nil),
		},
	}
	
	for i, tc := range testCases {
		for _,addr := range hostsAddress {
			tc.appServers = append(tc.appServers,testutil.NewRPCServer(t, addr,testutil.BoltWeight))
		}
		
		tc.appServer = testutil.NewRPCServer(t, hostsAddress[0],testutil.BoltWeight)
		
		t.Logf("start case #%d\n", i)
		tc.Start()
		go tc.RunCase(100)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d weighted proxy test failed, protocol: %s, error: %v\n", i, tc.AppProtocol, err)
			}
		case <-time.After(100 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d weighted proxy hang, protocol: %s\n", i, tc.AppProtocol)
		}
		close(tc.stop)
		time.Sleep(time.Second)
	}
}
