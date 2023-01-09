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
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
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
func testWRRLBCase(t *testing.T, hostNum int, weightFunc func(int) int) {
	pool := makePool(hostNum)
	hosts := []types.Host{}
	allWeight := 0.0
	for i := 0; i < hostNum; i++ {
		host := &mockHost{
			addr: pool.Get(),
			w:    uint32(weightFunc(i)), // 1-4
		}
		hosts = append(hosts, host)
		allWeight += float64(host.w)
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
		for i := 0; i < hostNum; i++ {
			addr := hosts[i].AddressString()
			rate := float64(results[addr]) / float64(subTotal)
			expected := edfFixedWeight(float64(weightFunc(i))) / allWeight
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
	type args struct {
		hostNum    int
		weightFunc func(int) int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "normal",
			args: args{
				hostNum: 4,
				weightFunc: func(i int) int {
					return i + 1
				},
			},
		},
		{
			name: "normal",
			args: args{
				hostNum: 4,
				weightFunc: func(i int) int {
					return i
				},
			},
		},
		{
			name: "all host weight are equal",
			args: args{
				hostNum: 10,
				weightFunc: func(i int) int {
					return 1
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testWRRLBCase(t, tt.args.hostNum, tt.args.weightFunc)
		})
	}
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

	hosts.Range(func(host types.Host) bool {
		mockRequest(host, true, 10)
		return true
	})
	// new lb to refresh edf
	balancer = NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveRequest}, hosts)
	actual := balancer.ChooseHost(newMockLbContext(nil))
	assert.NotNil(t, actual)

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
	hostSet := getMockHostSet(10)
	clusterInfo := getMockClusterInfo()
	lb := newMaglevLoadBalancer(clusterInfo, hostSet)

	testProtocol := types.ProtocolName("SomeProtocol")
	policy := &mockPolicy{}
	mockRoute := &mockRoute{
		routeRule: &mockRouteRule{
			policy: policy,
		},
	}
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, testProtocol)
	lbctx := &mockLbContext{
		context: ctx,
		route:   mockRoute,
	}
	// test empty hash policy return nil host
	host := lb.ChooseHost(lbctx)
	assert.Nil(t, host, "host should be nil")

	// test return non nil host
	policy.hashPolicy = &mockHashPolicy{}
	host = lb.ChooseHost(lbctx)
	// test 0 hash generated by mock route get 9 indexed host
	assert.Equalf(t, "127.0.0.9", host.AddressString(),
		"chosen host address string should be 127.0.0.9")
	host = lb.ChooseHost(lbctx)
	assert.Equalf(t, "127.0.0.0", host.AddressString(),
		"chosen host address string should be 127.0.0.0")
}

func Test_maglevLoadBalancerFallback(t *testing.T) {
	hostSet := getMockHostSet(10)

	mgv := newMaglevLoadBalancer(nil, hostSet)

	// set host-8 unhealthy
	hostSet.hosts[8].SetHealthFlag(api.FAILED_ACTIVE_HC)
	h := hostSet.hosts[8].Health()
	if !assert.Falsef(t, h, "Health() should be false") {
		t.FailNow()
	}

	host, _ := mgv.(*maglevLoadBalancer).chooseHostFromHostList(8)
	if !assert.Equalf(t, "host-9", host.Hostname(), "host name should be 'host-9'") {
		t.FailNow()
	}

	// set host-0, host1 unhealthy
	hostSet.hosts[9].SetHealthFlag(api.FAILED_ACTIVE_HC)
	hostSet.hosts[0].SetHealthFlag(api.FAILED_ACTIVE_HC)

	host, _ = mgv.(*maglevLoadBalancer).chooseHostFromHostList(8)
	if !assert.Equalf(t, "host-1", host.Hostname(), "host name should be 'host-1'") {
		t.FailNow()
	}

	// test all host is checked when fallback
	// create a new host set first
	hostSet = getMockHostSet(10)
	// leave only host[9] healthy
	for i := 2; i < 10; i++ {
		hostSet.hosts[i].SetHealthFlag(api.FAILED_ACTIVE_HC)
	}
	mgv = newMaglevLoadBalancer(nil, hostSet)
	host, _ = mgv.(*maglevLoadBalancer).chooseHostFromHostList(2)
	// host-9 will finally be chosen
	assert.Equalf(t, "host-1", host.Hostname(), "host name should be 'host-1'")
	// assert other 9 hosts is checked healthy
	assert.Equalf(t, 10, hostSet.healthCheckVisitedCount, "host name should be 'host-1'")
}
func getMockClusterInfo() *mockClusterInfo {
	return &mockClusterInfo{
		name: "mockClusterInfo",
	}
}

func TestReqRRChooseHost(t *testing.T) {
	hosts := createHostsetWithStats(exampleHostConfigs(), "test")
	balancer := NewLoadBalancer(&clusterInfo{lbType: types.RequestRoundRobin}, hosts)

	ctx := newMockLbContextWithCtx(nil, variable.NewVariableContext(context.Background()))
	// RR when use
	for i := 0; i < 2*hosts.Size(); i++ {
		ind := i % hosts.Size()
		host := balancer.ChooseHost(ctx)
		assert.Equal(t, host, hosts.allHosts[ind])
	}

	ctx0 := newMockLbContextWithCtx(nil, variable.NewVariableContext(context.Background()))
	ctx1 := newMockLbContextWithCtx(nil, variable.NewVariableContext(context.Background()))
	host := balancer.ChooseHost(ctx0)
	host1 := balancer.ChooseHost(ctx1)
	assert.Equal(t, host, host1)

}

func Test_roundRobinLoadBalancer_ChooseHost(t *testing.T) {
	type fields struct {
		hosts types.HostSet
		info  types.ClusterInfo
	}
	type args struct {
		context  types.LoadBalancerContext
		runCount int
	}
	type testCase struct {
		name           string
		fields         fields
		args           args
		want           map[string]int // map[host address]count
		wantUniformity float64
		resultChan     chan types.Host
	}
	// testcase1
	mockHosts_testcase1 := []types.Host{
		&mockHost{
			name:       "host1",
			addr:       "address1",
			healthFlag: GetHealthFlagPointer("address1"),
		},
		&mockHost{
			name:       "host2",
			addr:       "address2",
			healthFlag: GetHealthFlagPointer("address2"),
		},
		&mockHost{
			name:       "host3",
			addr:       "address3",
			healthFlag: GetHealthFlagPointer("address3"),
		},
		&mockHost{
			name:       "host4",
			addr:       "address4",
			healthFlag: GetHealthFlagPointer("address4"),
		},
	}
	mockHosts_testcase1[2].SetHealthFlag(api.FAILED_ACTIVE_HC)
	mockHosts_testcase1[0].SetHealthFlag(api.FAILED_ACTIVE_HC)
	hostSet1_testCase1 := &hostSet{}
	hostSet1_testCase1.setFinalHost(mockHosts_testcase1)
	tests := []testCase{
		{
			name: "LB-From4Hosts-2-healthy-2-unhealthy",
			fields: fields{
				info:  nil,
				hosts: hostSet1_testCase1,
			},
			args: args{
				context:  nil,
				runCount: 1000,
			},
			want: map[string]int{
				"address1": 0,
				"address2": 500,
				"address3": 0,
				"address4": 500,
			},
			wantUniformity: 0.0001,
			resultChan:     make(chan types.Host, 1000),
		},
	}
	// testcase2
	mockHosts_testcase2 := []types.Host{
		&mockHost{
			name:       "host-1",
			addr:       "address-1",
			healthFlag: GetHealthFlagPointer("address-1"),
		},
		&mockHost{
			name:       "host-2",
			addr:       "address-2",
			healthFlag: GetHealthFlagPointer("address-2"),
		},
		&mockHost{
			name:       "host-3",
			addr:       "address-3",
			healthFlag: GetHealthFlagPointer("address-3"),
		},
		&mockHost{
			name:       "host-4",
			addr:       "address-4",
			healthFlag: GetHealthFlagPointer("address-4"),
		},
	}
	mockHosts_testcase2[2].SetHealthFlag(api.FAILED_ACTIVE_HC)
	mockHosts_testcase2[1].SetHealthFlag(api.FAILED_ACTIVE_HC)
	mockHosts_testcase2[0].SetHealthFlag(api.FAILED_ACTIVE_HC)
	hostSet1_testCase2 := &hostSet{}
	hostSet1_testCase2.setFinalHost(mockHosts_testcase2)
	tests = append(tests, testCase{
		name: "LB-From4Hosts-1-healthy-3-unhealthy",
		fields: fields{
			info:  nil,
			hosts: hostSet1_testCase2,
		},
		args: args{
			context:  nil,
			runCount: 1000,
		},
		want: map[string]int{
			"address-1": 0,
			"address-2": 0,
			"address-3": 0,
			"address-4": 1000,
		},
		wantUniformity: 0.0,
		resultChan:     make(chan types.Host, 1000),
	})
	// testcase3
	mockHosts_testcase3 := []types.Host{
		&mockHost{
			name:       "host--1",
			addr:       "address--1",
			healthFlag: GetHealthFlagPointer("address--1"),
		},
		&mockHost{
			name:       "host--2",
			addr:       "address--2",
			healthFlag: GetHealthFlagPointer("address--2"),
		},
		&mockHost{
			name:       "host--3",
			addr:       "address--3",
			healthFlag: GetHealthFlagPointer("address--3"),
		},
		&mockHost{
			name:       "host--4",
			addr:       "address--4",
			healthFlag: GetHealthFlagPointer("address--4"),
		},
		&mockHost{
			name:       "host--5",
			addr:       "address--5",
			healthFlag: GetHealthFlagPointer("address--5"),
		},
		&mockHost{
			name:       "host--6",
			addr:       "address--6",
			healthFlag: GetHealthFlagPointer("address--6"),
		},
		&mockHost{
			name:       "host--7",
			addr:       "address--7",
			healthFlag: GetHealthFlagPointer("address--7"),
		},
		&mockHost{
			name:       "host--8",
			addr:       "address--8",
			healthFlag: GetHealthFlagPointer("address--8"),
		},
		&mockHost{
			name:       "host--9",
			addr:       "address--9",
			healthFlag: GetHealthFlagPointer("address--9"),
		},
	}
	mockHosts_testcase3[8].SetHealthFlag(api.FAILED_ACTIVE_HC)
	mockHosts_testcase3[7].SetHealthFlag(api.FAILED_ACTIVE_HC)
	mockHosts_testcase3[5].SetHealthFlag(api.FAILED_ACTIVE_HC)
	mockHosts_testcase3[4].SetHealthFlag(api.FAILED_ACTIVE_HC)
	mockHosts_testcase3[2].SetHealthFlag(api.FAILED_ACTIVE_HC)
	mockHosts_testcase3[1].SetHealthFlag(api.FAILED_ACTIVE_HC)
	hostSet1_testCase3 := &hostSet{}
	hostSet1_testCase3.setFinalHost(mockHosts_testcase3)
	tests = append(tests, testCase{
		name: "LB-From9Hosts-3-healthy-6-unhealthy",
		fields: fields{
			info:  nil,
			hosts: hostSet1_testCase3,
		},
		args: args{
			context:  nil,
			runCount: 1500,
		},
		want: map[string]int{
			"address--1": 500,
			"address--2": 0,
			"address--3": 0,
			"address--4": 500,
			"address--5": 0,
			"address--6": 0,
			"address--7": 500,
			"address--8": 0,
			"address--9": 0,
		},
		wantUniformity: 0.0001,
		resultChan:     make(chan types.Host, 1500),
	})
	// testcase4
	mockHosts_testcase4 := []types.Host{
		&mockHost{
			name:       "host--1",
			addr:       "address--1",
			healthFlag: GetHealthFlagPointer("address--1"),
		},
		&mockHost{
			name:       "host--2",
			addr:       "address--2",
			healthFlag: GetHealthFlagPointer("address--2"),
		},
		&mockHost{
			name:       "host--3",
			addr:       "address--3",
			healthFlag: GetHealthFlagPointer("address--3"),
		},
		&mockHost{
			name:       "host--4",
			addr:       "address--4",
			healthFlag: GetHealthFlagPointer("address--4"),
		},
		&mockHost{
			name:       "host--5",
			addr:       "address--5",
			healthFlag: GetHealthFlagPointer("address--5"),
		},
		&mockHost{
			name:       "host--6",
			addr:       "address--6",
			healthFlag: GetHealthFlagPointer("address--6"),
		},
		&mockHost{
			name:       "host--7",
			addr:       "address--7",
			healthFlag: GetHealthFlagPointer("address--7"),
		},
		&mockHost{
			name:       "host--8",
			addr:       "address--8",
			healthFlag: GetHealthFlagPointer("address--8"),
		},
		&mockHost{
			name:       "host--9",
			addr:       "address--9",
			healthFlag: GetHealthFlagPointer("address--9"),
		},
	}

	hostSet1_testCase4 := &hostSet{}
	hostSet1_testCase4.setFinalHost(mockHosts_testcase4)
	tests = append(tests, testCase{
		name: "LB-From9Hosts-9-healthy-0-unhealthy",
		fields: fields{
			info:  nil,
			hosts: hostSet1_testCase4,
		},
		args: args{
			context:  nil,
			runCount: 900,
		},
		want: map[string]int{
			"address--1": 100,
			"address--2": 100,
			"address--3": 100,
			"address--4": 100,
			"address--5": 100,
			"address--6": 100,
			"address--7": 100,
			"address--8": 100,
			"address--9": 100,
		},
		wantUniformity: 0.0,
		resultChan:     make(chan types.Host, 900),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := lbFactories[types.RoundRobin]
			lb := factory(tt.fields.info, tt.fields.hosts)
			defer func() {
				close(tt.resultChan)
				tt.fields.hosts.Range(func(host types.Host) bool {
					host.ClearHealthFlag(api.FAILED_ACTIVE_HC)
					return true
				})
			}()
			for l := 0; l < tt.args.runCount; l++ {
				go func() {
					gotHost := lb.ChooseHost(tt.args.context)
					tt.resultChan <- gotHost
				}()
			}

			for l := 0; l < tt.args.runCount; l++ {
				gotHost := <-tt.resultChan
				if gotHost == nil {
					t.Errorf("return nil host")
				}
				if chooseCount, ok := tt.want[gotHost.AddressString()]; ok {
					tt.want[gotHost.AddressString()] = chooseCount - 1
				}
			}

			deviationCount := 0
			for address, count := range tt.want {
				t.Logf("roundRobinLoadBalancer choose host count, address: %v, count: %v", address, count)
				if count > 0 {
					deviationCount += count
				}
			}

			if gotUniformity := float64(deviationCount) / float64(tt.args.runCount); gotUniformity > tt.wantUniformity {
				t.Errorf("roundRobinLoadBalancer choose host count err, gotUniformity: %v, wantUniformity: %v", gotUniformity, tt.wantUniformity)
			}
		})
	}
}

func TestWRRLoadBalancer(t *testing.T) {
	testCases := []struct {
		name                string
		hosts               []types.Host
		unhealthHostIndexes []int
		want                types.Host
	}{
		{
			name: "unhealthy-host-with-a-high-weight",
			hosts: []types.Host{
				&mockHost{addr: "192.168.1.1", w: 100},
				&mockHost{addr: "192.168.1.2", w: 1},
			},
			unhealthHostIndexes: []int{0},
			want:                &mockHost{addr: "192.168.1.2", w: 1},
		},
		{
			name: "without-healthy-hosts",
			hosts: []types.Host{
				&mockHost{addr: "192.168.1.1", w: 100},
				&mockHost{addr: "192.168.1.2", w: 1},
			},
			unhealthHostIndexes: []int{0, 1},
			want:                nil,
		},
	}

	for _, tc := range testCases {
		for _, index := range tc.unhealthHostIndexes {
			tc.hosts[index].SetHealthFlag(api.FAILED_ACTIVE_HC)
		}
		hs := &hostSet{}
		hs.setFinalHost(tc.hosts)
		lb := newWRRLoadBalancer(nil, hs)
		var h types.Host
		// 3 is the times of retrying to choose host in cluster manager.
		for i := 0; i < 3; i++ {
			h = lb.ChooseHost(nil)
			if h == nil {
				continue
			}
		}

		if h == tc.want {
			continue
		}

		if h == nil {
			t.Fatalf("case:%s, expected %s, but got a nil host", tc.name, tc.want.AddressString())
		} else if h.AddressString() != tc.want.AddressString() {
			t.Fatalf("case:%s, expected %s, but got: %s", tc.name, tc.want.AddressString(), h.AddressString())
		}
	}
}

func TestNewLACBalancer(t *testing.T) {
	balancer := NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveConnection}, &hostSet{})
	assert.NotNil(t, balancer)
	assert.IsType(t, &leastActiveConnectionLoadBalancer{}, balancer)
}
