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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

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

func createHealthCheckCluster(servers []*healthCheckTestServer) types.Cluster {
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
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy: 2,
			SubsetSelectors: [][]string{
				[]string{
					"version",
				},
			},
		},
	}
	cluster := NewCluster(clusterConfig)
	var hosts []types.Host
	for _, s := range servers {
		hostConfig := v2.Host{
			HostConfig: v2.HostConfig{
				Address: s.hostConfig.Address,
			},
			MetaData: api.Metadata{
				"version": "1.0.0",
			},
		}
		hosts = append(hosts, NewSimpleHost(hostConfig, cluster.Snapshot().ClusterInfo()))
	}
	cluster.UpdateHosts(NewHostSet(hosts))
	return cluster
}

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
	// choose fallback
	for i := 0; i < 100; i++ {
		host := cluster.Snapshot().LoadBalancer().ChooseHost(nil)
		results[host.AddressString()] = results[host.AddressString()] + 1
	}
	// choose policy
	ctx := newMockLbContext(map[string]string{
		"version": "1.0.0",
	})
	for i := 0; i < 100; i++ {
		host := cluster.Snapshot().LoadBalancer().ChooseHost(ctx)
		results[host.AddressString()] = results[host.AddressString()] + 1
	}
	// verify
	for addr, count := range results {
		if count == 0 {
			t.Errorf("host %s is not be choosed", addr)
		}
	}
	// close a server and waits
	testServers[0].server.Close()
	time.Sleep(2 * time.Second)
	// choose all hosts randomly, unhealthy server should not be choosed
	// choose fallback
	for i := 0; i < 100; i++ {
		host := cluster.Snapshot().LoadBalancer().ChooseHost(nil)
		if host == nil || host.AddressString() == testServers[0].hostConfig.Address {
			t.Fatal("choose an unhealthy host")
		}
	}
	// choose policy
	for i := 0; i < 100; i++ {
		host := cluster.Snapshot().LoadBalancer().ChooseHost(ctx)
		if host == nil || host.AddressString() == testServers[0].hostConfig.Address {
			t.Fatal("choose an unhealthy host")
		}
	}
	// clear result, restart server and waits, new server will be choosed again
	for addr := range results {
		results[addr] = 0
	}
	srv := testServers[0].restart(t)
	defer srv.Close()
	time.Sleep(2 * time.Second)
	for i := 0; i < 100; i++ {
		host := cluster.Snapshot().LoadBalancer().ChooseHost(nil)
		results[host.AddressString()] = results[host.AddressString()] + 1
	}
	for i := 0; i < 100; i++ {
		host := cluster.Snapshot().LoadBalancer().ChooseHost(ctx)
		results[host.AddressString()] = results[host.AddressString()] + 1
	}
	// verify
	for addr, count := range results {
		if count == 0 {
			t.Errorf("host %s is not be choosed", addr)
		}
	}
	if cbTest.ToHealthy != 1 || cbTest.ToUnhealthy != 1 {
		t.Errorf("additional callbacks is not called expected %v", cbTest)
	}
	// stop health check
	cluster.(*simpleCluster).healthChecker.Stop()
}

// If a cluster adds/remove hosts via admin api, the health check should know it
func TestHealthCheckWithDynamicCluster(t *testing.T) {
	var testServers []*healthCheckTestServer
	results := make(map[string]int)
	for i := 0; i < 3; i++ {
		s := newHealthCheckTestServer()
		testServers = append(testServers, s)
		results[s.hostConfig.Address] = 0
	}
	log.DefaultLogger.SetLogLevel(log.INFO)
	defer log.DefaultLogger.SetLogLevel(log.ERROR)
	cluster := createHealthCheckCluster(testServers)
	// choose host and add new host concurrency
	// new host should be choosed after add
	mux := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for i := 0; i < 100; i++ {
			host := cluster.Snapshot().LoadBalancer().ChooseHost(nil)
			func() {
				mux.Lock()
				defer mux.Unlock()
				results[host.AddressString()] = results[host.AddressString()] + 1
			}()
			time.Sleep(50 * time.Millisecond)
		}
		wg.Done()
	}()
	ctx := newMockLbContext(map[string]string{
		"version": "1.0.0",
	})
	wg.Add(1)
	go func() {
		for i := 0; i < 100; i++ {
			host := cluster.Snapshot().LoadBalancer().ChooseHost(ctx)
			func() {
				mux.Lock()
				defer mux.Unlock()
				results[host.AddressString()] = results[host.AddressString()] + 1
			}()
			time.Sleep(50 * time.Millisecond)
		}
		wg.Done()
	}()
	snew := newHealthCheckTestServer()
	func() {
		mux.Lock()
		defer mux.Unlock()
		results[snew.hostConfig.Address] = 0
	}()
	go func() {
		var hosts []types.Host
		cluster.Snapshot().HostSet().Range(func(host types.Host) bool {
			hosts = append(hosts, host)
			return true
		})
		newConfig := v2.Host{
			HostConfig: v2.HostConfig{
				Address: snew.hostConfig.Address,
			},
			MetaData: api.Metadata{
				"version": "1.0.0",
			},
		}
		hosts = append(hosts, NewSimpleHost(newConfig, cluster.Snapshot().ClusterInfo()))
		cluster.UpdateHosts(NewHostSet(hosts))
	}()
	wg.Wait()
	// verify all hosts should be choosed
	for addr, count := range results {
		if count == 0 {
			t.Errorf("host %s is not be choosed", addr)
		}
	}
	// the new host should have health check
	// we try to close the new server, the health check should know it
	snew.server.Close()
	time.Sleep(2 * time.Second)
	for i := 0; i < 100; i++ {
		host := cluster.Snapshot().LoadBalancer().ChooseHost(nil)
		if host.AddressString() == snew.hostConfig.Address {
			t.Fatal("choose a unhealthy host:", host.Health())
		}
	}
	// choose policy
	for i := 0; i < 100; i++ {
		host := cluster.Snapshot().LoadBalancer().ChooseHost(ctx)
		if host.AddressString() == snew.hostConfig.Address {
			t.Fatal("choose an unhealthy host:", host.Health())
		}
	}
	var delHosts []types.Host // host after deleted
	removed := cluster.Snapshot().HostSet().Get(0)
	delHosts = append(delHosts, listHostSet(cluster.Snapshot().HostSet())[1:]...)
	cluster.UpdateHosts(NewHostSet(delHosts))
	// clear results
	for addr := range results {
		results[addr] = 0
	}
	// choose fallback
	for i := 0; i < 100; i++ {
		host := cluster.Snapshot().LoadBalancer().ChooseHost(nil)
		if host == nil {
			t.Fatal("cannot found a healthy host")
		}
		results[host.AddressString()] = results[host.AddressString()] + 1
	}
	// choose policy
	for i := 0; i < 100; i++ {
		host := cluster.Snapshot().LoadBalancer().ChooseHost(ctx)
		if host == nil {
			t.Fatal("cannot found a healthy host")
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
	// stop health checker
	cluster.(*simpleCluster).healthChecker.Stop()
}

func listHostSet(hs types.HostSet) []types.Host {
	ret := make([]types.Host, 0, hs.Size())
	hs.Range(func(host types.Host) bool {
		ret = append(ret, host)
		return true
	})
	return ret
}
