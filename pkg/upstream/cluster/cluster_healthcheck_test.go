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

package cluster

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// Test a cluster with a healt check
// use default tcp dial, upstream use httptest server, just for create a connection
func TestClusterHealthCheck(t *testing.T) {
	// init servers in upstream
	var testServers []*healthCheckTestServer
	results := make(map[string]int)
	for i := 0; i < 3; i++ {
		s := newHealthCheckTestServer()
		testServers = append(testServers, s)
		results[s.hostConfig.Address] = 0
	}
	cluster := createHealthCheckCluster(testServers)
	cbTest := &mockCbServer{}
	cluster.AddHealthCheckCallbacks(cbTest.Record)
	// case 1
	// choose all hosts randomly
	for i := 0; i < 100; i++ {
		host := cluster.Info().LBInstance().ChooseHost(nil)
		if host == nil {
			t.Fatal("no host is choosed")
		}
		results[host.AddressString()] = results[host.AddressString()] + 1
	}
	// verify all hosts should be choosed
	for addr, count := range results {
		if count == 0 {
			t.Errorf("host %s is not be choosed", addr)
		}
	}
	// case 2
	// close a server and waits
	testServers[0].server.Close()
	time.Sleep(time.Second)
	// choose all hosts randomly, unhealthy server should not be choosed
	for i := 0; i < 100; i++ {
		host := cluster.Info().LBInstance().ChooseHost(nil)
		if host == nil {
			t.Fatal("no host is choosed")
		}
		if host.AddressString() == testServers[0].hostConfig.Address {
			t.Fatal("choose a unhealthy host")
		}
	}
	// case 3
	// restart the server and waits, will choose again
	// httptest server not support restart, so we listen the port by ourselves
	srv := testServers[0].restart(t)
	defer srv.Close()
	time.Sleep(2 * time.Second)
	// clear results
	for addr := range results {
		results[addr] = 0
	}
	// choose all hosts randomly
	for i := 0; i < 100; i++ {
		host := cluster.Info().LBInstance().ChooseHost(nil)
		if host == nil {
			t.Fatal("no host is choosed")
		}
		results[host.AddressString()] = results[host.AddressString()] + 1
	}
	// verify all hosts should be choosed
	for addr, count := range results {
		if count == 0 {
			t.Errorf("host %s is not be choosed", addr)
		}
	}
	if cbTest.ToHealthy != 1 || cbTest.ToUnhealthy != 1 {
		t.Errorf("additional callbacks is not called expected %v", cbTest)
	}
}

// If a cluster adds/remove hosts via admin api, the health check should know it
func TestHealthCheckWithDynamicCluster(t *testing.T) {
	// init servers in upstream
	var testServers []*healthCheckTestServer
	results := make(map[string]int)
	for i := 0; i < 3; i++ {
		s := newHealthCheckTestServer()
		testServers = append(testServers, s)
		results[s.hostConfig.Address] = 0
	}
	cluster := createHealthCheckCluster(testServers)
	// case 1
	// add a host
	raw := cluster.(*simpleInMemCluster)
	var hosts []types.Host
	for _, s := range testServers {
		hosts = append(hosts, NewHost(s.hostConfig, cluster.Info()))
	}
	snew := newHealthCheckTestServer()
	hosts = append(hosts, NewHost(snew.hostConfig, cluster.Info()))
	raw.UpdateHosts(hosts)
	results[snew.hostConfig.Address] = 0
	time.Sleep(time.Second)
	// choose all hosts randomly
	for i := 0; i < 100; i++ {
		host := cluster.Info().LBInstance().ChooseHost(nil)
		if host == nil {
			t.Fatal("no host is choosed")
		}
		results[host.AddressString()] = results[host.AddressString()] + 1
	}
	// verify all hosts should be choosed
	for addr, count := range results {
		if count == 0 {
			t.Errorf("host %s is not be choosed", addr)
		}
	}
	// case 2
	// the new host should have health check
	// we try to close the new server, the health check should know it
	snew.server.Close()
	time.Sleep(time.Second)
	// choose all hosts randomly, unhealthy server should not be choosed
	for i := 0; i < 100; i++ {
		host := cluster.Info().LBInstance().ChooseHost(nil)
		if host == nil {
			t.Fatal("no host is choosed")
		}
		if host.AddressString() == snew.hostConfig.Address {
			t.Fatal("choose a unhealthy host")
		}
	}
	// case 3
	// remove a host
	removed := hosts[0]
	hosts = append(hosts[:0], hosts[1:]...)
	raw.UpdateHosts(hosts)
	time.Sleep(time.Second)
	// clear results
	for addr := range results {
		results[addr] = 0
	}
	// choose all hosts randomly
	for i := 0; i < 100; i++ {
		host := cluster.Info().LBInstance().ChooseHost(nil)
		if host == nil {
			t.Fatal("no host is choosed")
		}
		results[host.AddressString()] = results[host.AddressString()] + 1
	}
	// the removed one should not not be choosed
	// the closed server should also not be choosed
	for addr, count := range results {
		switch addr {
		case removed.AddressString(), snew.hostConfig.Address: // removed server and closed server
			if count != 0 {
				t.Errorf("choose server %s, but should not", addr)
			}
		default: // healthy server
			if count == 0 {
				t.Errorf("host %s is not be choosed", addr)
			}
		}
	}
}

//
type healthCheckTestServer struct {
	server     *httptest.Server
	hostConfig v2.Host
}

func newHealthCheckTestServer() *healthCheckTestServer {
	s := httptest.NewServer(nil)
	addr := strings.Split(s.URL, "http://")[1]
	cfg := v2.Host{
		HostConfig: v2.HostConfig{
			Address: addr,
		},
	}
	return &healthCheckTestServer{
		server:     s,
		hostConfig: cfg,
	}
}

func (s *healthCheckTestServer) restart(t *testing.T) *http.Server {
	srv := &http.Server{
		Addr: s.hostConfig.Address,
	}
	go srv.ListenAndServe()
	return srv
}

func createHealthCheckCluster(servers []*healthCheckTestServer) types.Cluster {
	// cluster config
	clusterConfig := v2.Cluster{
		Name:        "test",
		ClusterType: v2.SIMPLE_CLUSTER,
		LbType:      v2.LB_RANDOM,
		HealthCheck: v2.HealthCheck{
			HealthCheckConfig: v2.HealthCheckConfig{
				ServiceName:        "test",
				HealthyThreshold:   1,
				UnhealthyThreshold: 1,
			},
			Interval: 500 * time.Millisecond,
		},
	}
	cluster := NewCluster(clusterConfig, nil, true)
	// Add Hosts, which is called in clustermanager->primaryCluster->simpleInMemCluster
	raw := cluster.(*simpleInMemCluster)
	var hosts []types.Host
	for _, s := range servers {
		hosts = append(hosts, NewHost(s.hostConfig, cluster.Info()))
	}
	raw.UpdateHosts(hosts)
	return cluster
}

// addtional callbacks
type mockCbServer struct {
	ToUnhealthy uint32
	ToHealthy   uint32
}

func (s *mockCbServer) Record(host types.Host, changed bool, isHealthy bool) {
	if changed {
		if isHealthy {
			atomic.AddUint32(&s.ToHealthy, 1)
		} else {
			atomic.AddUint32(&s.ToUnhealthy, 1)
		}
	}
}
