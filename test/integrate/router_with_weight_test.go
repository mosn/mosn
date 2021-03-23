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
	"math"
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"

	testutil "mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

type weightCase struct {
	*XTestCase
	appServers map[string]testutil.UpstreamServer
	clusters   []*testutil.WeightCluster
}

// support SofaRPC with xprotocol only
func newWeightCase(t *testing.T, clusters []*testutil.WeightCluster) *weightCase {
	tc := NewXTestCase(t, bolt.ProtocolName, testutil.NewRPCServer(t, "", bolt.ProtocolName))
	appServers := make(map[string]testutil.UpstreamServer)
	for _, cluster := range clusters {
		for _, host := range cluster.Hosts {
			appServers[host.Addr] = testutil.NewRPCServer(t, host.Addr, bolt.ProtocolName)
		}
	}
	return &weightCase{
		XTestCase:  tc,
		appServers: appServers,
		clusters:   clusters,
	}
}

func (c *weightCase) Start() {
	for _, appserver := range c.appServers {
		appserver.GoServe()
	}

	meshAddr := testutil.CurrentMeshAddr()
	c.ClientMeshAddr = meshAddr
	// use bolt as example
	cfg := testutil.CreateXWeightProxyMesh(meshAddr, bolt.ProtocolName, c.clusters)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		for _, appserver := range c.appServers {
			appserver.Close()
		}
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

const thres float64 = 0.3

func (c *weightCase) Verify(t *testing.T) {
	clusterCounts := []uint32{}
	clusterTotalCount := uint32(0)
	for index, cluster := range c.clusters {
		hostWeights := []uint32{}
		hostCounts := []uint32{}
		totalWeight := uint32(0)
		totalCount := uint32(0)
		for _, host := range cluster.Hosts {
			hostWeights = append(hostWeights, host.Weight)
			count := c.appServers[host.Addr].(*testutil.RPCServer).Count
			hostCounts = append(hostCounts, count)
			totalWeight += host.Weight
			totalCount += count
		}
		// compare count percent and weight percent, in range 'thres'
		for i := range cluster.Hosts {
			ratio := float64(hostCounts[i]) / float64(totalCount)
			expected := float64(hostWeights[i]) / float64(totalWeight)
			if math.Abs(ratio-expected) > thres {
				t.Errorf("cluster #%d weighted host selected error, host#%d expected: %f, got: %f", index, i, expected, ratio)
			}
			t.Logf("cluster #%d weighted host #%d expected: %f, got: %f", index, i, expected, ratio)
		}
		// cluster info
		clusterCounts = append(clusterCounts, totalCount)
		clusterTotalCount += totalCount
	}
	for i := range c.clusters {
		// cluster total weight should be 100
		expected := float64(c.clusters[i].Weight) / 100.0
		ratio := float64(clusterCounts[i]) / float64(clusterTotalCount)
		if math.Abs(ratio-expected) > thres {
			t.Errorf("cluster #%d weighted cluster selected error,  expected: %f, got: %f", i, expected, ratio)
		}
		t.Logf("cluster #%d weighted expected: %f, got: %f", i, expected, ratio)
	}
}

func TestWeightProxy(t *testing.T) {
	hostsAddress := []string{"127.0.0.1:8081", "127.0.0.1:8082", "127.0.0.1:8083", "127.0.0.1:8084"}
	cluster1 := &testutil.WeightCluster{
		Name:   "cluster1",
		Weight: 40,
		Hosts: []*testutil.WeightHost{
			{Addr: hostsAddress[0], Weight: 1},
			{Addr: hostsAddress[1], Weight: 2},
		},
	}
	cluster2 := &testutil.WeightCluster{
		Name:   "cluster2",
		Weight: 60,
		Hosts: []*testutil.WeightHost{
			{Addr: hostsAddress[2], Weight: 1},
			{Addr: hostsAddress[3], Weight: 2},
		},
	}
	tc := newWeightCase(t, []*testutil.WeightCluster{cluster1, cluster2})
	pass := false
	tc.Start()
	go tc.RunCase(100, 0)
	select {
	case err := <-tc.C:
		if err != nil {
			t.Errorf("[ERROR MESSAGE] weighted proxy test failed, error: %v\n", err)
		}
		pass = true
	case <-time.After(100 * time.Second):
		t.Error("[ERROR MESSAGE] weighted proxy hangn")
	}
	tc.FinishCase()
	// Verify Weight
	if pass {
		tc.Verify(t)
	}
}
