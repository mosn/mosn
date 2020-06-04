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
	"context"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

// load should be balanced when node fails
func TestRandomLBWhenNodeFailBalanced(t *testing.T) {
	defer func() {
		// clear healthStore
		healthStore = sync.Map{}
	}()

	pool := makePool(4)
	var hosts []types.Host
	var unhealthyIdx = 2
	for i := 0; i < 4; i++ {
		host := &mockHost{
			addr: pool.Get(),
		}
		if i == unhealthyIdx {
			host.SetHealthFlag(api.FAILED_ACTIVE_HC)
		}
		hosts = append(hosts, host)
	}

	hs := &hostSet{}
	hs.setFinalHost(hosts)
	lb := newRandomLoadBalancer(nil, hs)
	total := 1000000
	runCase := func(subTotal int) {
		results := map[string]int{}
		for i := 0; i < subTotal; i++ {
			h := lb.ChooseHost(nil)
			v, ok := results[h.AddressString()]
			if !ok {
				v = 0
			}
			results[h.AddressString()] = v + 1
		}
		for i := 0; i < 4; i++ {
			addr := hosts[i].AddressString()
			rate := float64(results[addr]) / float64(subTotal)
			expected := 0.33333
			if i == unhealthyIdx {
				expected = 0.000
			}
			if math.Abs(rate-expected) > 0.1 { // no lock, have deviation 10% is acceptable
				t.Errorf("%s request rate is %f, expected %f", addr, rate, expected)
			}
			t.Logf("%s request rate is %f, request count: %d", addr, rate, results[addr])
		}
	}
	// simple test
	runCase(total)
	// concurr
	wg := sync.WaitGroup{}
	subTotal := total / 10
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			runCase(subTotal)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestWRRLB(t *testing.T) {
	pool := makePool(4)
	hosts := []types.Host{}
	for i := 0; i < 4; i++ {
		host := &mockHost{
			addr: pool.Get(),
			w:    uint32(i + 1), // 1-4
		}
		hosts = append(hosts, host)
	}
	// 1:2:3:4
	hs := &hostSet{}
	hs.setFinalHost(hosts)
	lb := newWRRLoadBalancer(nil, hs)
	total := 1000000
	runCase := func(subTotal int) {
		results := map[string]int{}
		for i := 0; i < subTotal; i++ {
			h := lb.ChooseHost(nil)
			v, ok := results[h.AddressString()]
			if !ok {
				v = 0
			}
			results[h.AddressString()] = v + 1
		}
		for i := 0; i < 4; i++ {
			addr := hosts[i].AddressString()
			rate := float64(results[addr]) / float64(subTotal)
			expected := float64(i+1) / 10.0
			if math.Abs(rate-expected) > 0.1 { // no lock, have deviation 10% is acceptable
				t.Errorf("%s request rate is %f, expected %f", addr, rate, expected)
			}
			t.Logf("%s request rate is %f, request count: %d", addr, rate, results[addr])
		}
	}
	// simple test
	runCase(total)
	// concurr
	wg := sync.WaitGroup{}
	subTotal := total / 10
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			runCase(subTotal)
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkWRRLbSimple(b *testing.B) {
	pool := makePool(4)
	hosts := []types.Host{}
	for i := 0; i < 4; i++ {
		host := &mockHost{
			addr: pool.Get(),
			w:    uint32(i + 1), // 1-4
		}
		hosts = append(hosts, host)
	}
	hs := &hostSet{}
	hs.setFinalHost(hosts)
	lb := newWRRLoadBalancer(nil, hs)
	b.Run("WRRSimple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			lb.ChooseHost(nil)
		}
	})

}

func BenchmarkWRRLbMultiple(b *testing.B) {
	testCases := []struct {
		name       string
		count      int
		max_weight int
	}{
		{
			name:       "WRR_10_100",
			count:      10,
			max_weight: 100,
		},
		{
			name:       "WRR_1000_100",
			count:      1000,
			max_weight: 100,
		},
		{
			name:       "WRR_1000_1000",
			count:      1000,
			max_weight: 1000,
		},
	}
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, tc := range testCases {
		pool := makePool(tc.count)
		hosts := []types.Host{}
		for i := 0; i < tc.count; i++ {
			host := &mockHost{
				addr: pool.Get(),
				w:    uint32(1 + rand.Intn(tc.max_weight)),
			}
			hosts = append(hosts, host)
		}
		hs := &hostSet{}
		hs.setFinalHost(hosts)
		lb := newWRRLoadBalancer(nil, hs)
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				lb.ChooseHost(nil)
			}
		})
	}
}

func BenchmarkWRRLbParallel(b *testing.B) {
	testCases := []struct {
		name       string
		count      int
		max_weight int
	}{
		{
			name:       "WRR_10_100_Parallel",
			count:      10,
			max_weight: 100,
		},
		{
			name:       "WRR_1000_100_Parallel",
			count:      1000,
			max_weight: 100,
		},
		{
			name:       "WRR_1000_1000_Parallel",
			count:      1000,
			max_weight: 1000,
		},
	}
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, tc := range testCases {
		pool := makePool(tc.count)
		hosts := []types.Host{}
		for i := 0; i < tc.count; i++ {
			host := &mockHost{
				addr: pool.Get(),
				w:    uint32(1 + rand.Intn(tc.max_weight)),
			}
			hosts = append(hosts, host)
		}
		hs := &hostSet{}
		hs.setFinalHost(hosts)
		lb := newWRRLoadBalancer(nil, hs)
		b.Run(tc.name, func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					lb.ChooseHost(nil)
				}
			})
		})
	}

}

func TestNewLARBalancer(t *testing.T) {
	balancer := NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveRequest}, &hostSet{})
	assert.NotNil(t, balancer)
	assert.IsType(t, &leastActiveRequestLoadBalancer{}, balancer)
}

func TestLARChooseHost(t *testing.T) {
	hosts := createHostsetWithStats(exampleHostConfigs(), "test")
	balancer := NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveRequest}, hosts)
	host := balancer.ChooseHost(newMockLbContext(nil))
	assert.NotNil(t, host)

	for _, host := range hosts.Hosts() {
		mockRequest(host, true, 10)
	}
	// new lb to refresh edf
	balancer = NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveRequest}, hosts)
	actual := balancer.ChooseHost(newMockLbContext(nil))
	assert.Equal(t, hosts.allHosts[6], actual)
	actual = balancer.ChooseHost(newMockLbContext(nil))

	// test only one host
	h := exampleHostConfigs()[0:1]
	hosts = createHostsetWithStats(h, "test")
	balancer = NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveRequest}, hosts)
	actual = balancer.ChooseHost(nil)
	assert.Equal(t, hosts.allHosts[0], actual)

	// test no host
	h = exampleHostConfigs()[0:0]
	hosts = createHostsetWithStats(h, "test")
	balancer = NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveRequest}, hosts)
	actual = balancer.ChooseHost(nil)
	assert.Nil(t, actual)

}

func mockRequest(host types.Host, active bool, times int) {
	for i := 0; i < times; i++ {
		if active {
			host.HostStats().UpstreamRequestActive.Inc(1)
		}
	}
}

func Test_maglevLoadBalancer(t *testing.T) {
	hostSet := getMockHostSet()
	clusterInfo := getMockClusterInfo()
	lb := newMaglevLoadBalancer(clusterInfo, hostSet)

	rc := &v2.RouterConfiguration{
		VirtualHosts: []*v2.VirtualHost{
			{
				Routers: []v2.Router{
					{RouterConfig: v2.RouterConfig{
						Route: v2.RouteAction{
							RouterActionConfig: v2.RouterActionConfig{
								ClusterName: "mockClusterInfo",
								HashPolicy: []v2.HashPolicy{
									{Header: &v2.HeaderHashPolicy{Key: "header_key"}},
								},
							},
						},
					}},
				},
			},
		},
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: "headerRouter",
		},
	}
	err := router.GetRoutersMangerInstance().AddOrUpdateRouters(rc)
	assert.NoErrorf(t, err, "add or update router failed, %+v", err)

	testProtocol := types.ProtocolName("SomeProtocol")
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyDownStreamProtocol, testProtocol)
	ctx = mosnctx.WithValue(ctx, types.ContextKeyProxyRouter, "headerRouter")

	lbctx := &mockLbContext{
		context: ctx,
	}

	headerGetter := func(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
		return "test_header_value", nil
	}
	cookieGetter := func(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
		return "test_cookie_value", nil
	}

	testFunc := func(expect []string) {
		hostsResult := []string{}
		// query 5 times
		for i := 0; i < 5; i++ {
			host := lb.ChooseHost(lbctx)
			hostsResult = append(hostsResult, host.Hostname())
		}
		if !reflect.DeepEqual(expect, hostsResult) {
			t.Errorf("hosts expect to be %+v, get %+v", expect, hostsResult)
			t.FailNow()
		}
	}

	// test header
	headerValue := variable.NewBasicVariable("SomeProtocol_request_header_", nil, headerGetter, nil, 0)
	variable.RegisterPrefixVariable(headerValue.Name(), headerValue)
	variable.RegisterProtocolResource(testProtocol, api.HEADER, types.VarProtocolRequestHeader)
	testFunc([]string{
		"host-8", "host-8", "host-8", "host-8", "host-8",
	})

	// test cookie
	cookieValue := variable.NewBasicVariable("SomeProtocol_cookie_", nil, cookieGetter, nil, 0)
	variable.RegisterPrefixVariable(cookieValue.Name(), cookieValue)
	variable.RegisterProtocolResource(testProtocol, api.COOKIE, types.VarProtocolCookie)
	rc.VirtualHosts[0].Routers[0].RouterConfig.Route.HashPolicy[0] = v2.HashPolicy{
		HttpCookie: &v2.HttpCookieHashPolicy{
			Name: "cookie_name",
			Path: "cookie_path",
			TTL:  api.DurationConfig{5 * time.Second},
		},
	}
	rc.RouterConfigName = "cookieRouter"
	lbctx.context = mosnctx.WithValue(ctx, types.ContextKeyProxyRouter, "cookieRouter")
	err = router.GetRoutersMangerInstance().AddOrUpdateRouters(rc)
	assert.NoErrorf(t, err, "add or update router failed, %+v", err)

	testFunc([]string{
		"host-0", "host-0", "host-0", "host-0", "host-0",
	})

	// test source IP
	rc.VirtualHosts[0].Routers[0].RouterConfig.Route.HashPolicy[0] = v2.HashPolicy{
		SourceIP: &v2.SourceIPHashPolicy{},
	}
	rc.RouterConfigName = "sourceIPRouter"
	lbctx.context = mosnctx.WithValue(ctx, types.ContextKeyProxyRouter, "sourceIPRouter")
	err = router.GetRoutersMangerInstance().AddOrUpdateRouters(rc)
	assert.NoErrorf(t, err, "add or update router failed, %+v", err)
	testFunc([]string{
		"host-8", "host-8", "host-8", "host-8", "host-8",
	})
}

func Test_segmentTreeFallback(t *testing.T) {
	hostSet := getMockHostSet()

	mgv := newMaglevLoadBalancer(nil, hostSet)

	// set host-8 unhealthy
	hostSet.hosts[8].SetHealthFlag(api.FAILED_ACTIVE_HC)
	h := hostSet.hosts[8].Health()
	if !assert.Falsef(t, h, "Health() should be false") {
		t.FailNow()
	}
	node, err := mgv.(*maglevLoadBalancer).fallbackSegTree.Leaf(8)
	if err != nil {
		t.Error(err)
	}
	mgv.(*maglevLoadBalancer).fallbackSegTree.Update(node)

	host := mgv.(*maglevLoadBalancer).chooseHostFromSegmentTree(8)
	if !assert.Equalf(t, "host-9", host.Hostname(), "host name should be 'host-9'") {
		t.FailNow()
	}

	// set host-9 unhealthy
	hostSet.hosts[9].SetHealthFlag(api.FAILED_ACTIVE_HC)
	h = hostSet.hosts[9].Health()
	if !assert.Falsef(t, h, "Health() should be false") {
		t.FailNow()
	}
	node, err = mgv.(*maglevLoadBalancer).fallbackSegTree.Leaf(9)
	if err != nil {
		t.Error(err)
	}
	mgv.(*maglevLoadBalancer).fallbackSegTree.Update(node)

	host = mgv.(*maglevLoadBalancer).chooseHostFromSegmentTree(8)
	if !assert.Equalf(t, "host-6", host.Hostname(), "host name should be 'host-6'") {
		t.FailNow()
	}
}

func getMockHostSet() *mockHostSet {
	hosts := []types.Host{}
	hostCount := 10
	for i := 0; i < hostCount; i++ {
		h := &mockHost{
			name: fmt.Sprintf("host-%d", i),
			addr: fmt.Sprintf("127.0.0.%d", i),
		}
		hosts = append(hosts, h)
	}
	return &mockHostSet{
		hosts: hosts,
	}
}

func getMockClusterInfo() *mockClusterInfo {
	return &mockClusterInfo{
		name: "mockClusterInfo",
	}
}
