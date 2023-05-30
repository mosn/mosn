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
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/cch123/supermonkey"
	"github.com/stretchr/testify/assert"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
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
			expected := fixHostWeight(float64(weightFunc(i))) / allWeight
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

func Test_slowStartDurationFactorFuncWithNowFunc(t *testing.T) {
	now := time.Now()
	host := &simpleHost{
		hostname:                "localhost",
		addressString:           "127.0.0.1:8080",
		weight:                  100,
		lastHealthCheckPassTime: now,
	}

	check := func(slowStart types.SlowStart, now time.Time, excepted float64) {
		nowFunc := func() time.Time {
			return now
		}
		f := slowStartDurationFactorFuncWithNowFunc(&clusterInfo{slowStart: slowStart}, host, nowFunc)
		assert.Equal(t, excepted, f)
	}

	check(types.SlowStart{SlowStartDuration: 10 * time.Second}, now, 0.1)
	check(types.SlowStart{SlowStartDuration: 10 * time.Second}, now.Add(5*time.Second), 0.5)
	check(types.SlowStart{SlowStartDuration: 10 * time.Second}, now.Add(10*time.Second), 1)
	check(types.SlowStart{SlowStartDuration: 10 * time.Second}, now.Add(20*time.Second), 1)
	check(types.SlowStart{SlowStartDuration: 20 * time.Second}, now.Add(5*time.Second), 0.25)

	// always return 1.0 if given non-positive duration
	check(types.SlowStart{SlowStartDuration: 0 * time.Second}, now, 1)
	check(types.SlowStart{SlowStartDuration: 0 * time.Second}, now.Add(5*time.Second), 1)

	check(types.SlowStart{SlowStartDuration: -1 * time.Second}, now, 1)
	check(types.SlowStart{SlowStartDuration: -1 * time.Second}, now.Add(5*time.Second), 1)
}

type notHostWeightItem struct {
	weight uint32
}

func (item notHostWeightItem) Weight() uint32 {
	return item.weight
}

func Test_slowStartHostWeightFunc(t *testing.T) {
	f := 1.0
	RegisterSlowStartMode("test", func(info types.ClusterInfo, host types.Host) float64 {
		return f
	})

	host := &simpleHost{
		hostname:      "localhost",
		addressString: "127.0.0.1:8080",
		weight:        100,
	}
	wi := notHostWeightItem{weight: 100}

	hostWeightFunc := func(host WeightItem) float64 {
		return float64(host.Weight())
	}

	check := func(host WeightItem, info types.ClusterInfo, except float64, funcShouldBeSame bool) {
		slowStartHostWeightFunc := slowStartHostWeightFunc(info, hostWeightFunc)
		if funcShouldBeSame {
			assert.Equal(t, reflect.ValueOf(hostWeightFunc).Pointer(), reflect.ValueOf(slowStartHostWeightFunc).Pointer())
		}
		w := slowStartHostWeightFunc(host)
		assert.Equal(t, except, w)
	}

	f = 0
	// host is not types.host, return directly
	check(wi, &clusterInfo{slowStart: types.SlowStart{
		Mode:             "test",
		Aggression:       v2.SlowStartDefaultAggression,
		MinWeightPercent: v2.SlowStartDefaultMinWeightPercent,
	}}, 100, false)

	f = 0.1
	check(host, nil, 100, true)

	check(host, &clusterInfo{slowStart: types.SlowStart{
		Mode:             "",
		Aggression:       v2.SlowStartDefaultAggression,
		MinWeightPercent: v2.SlowStartDefaultMinWeightPercent,
	}}, 100, true)

	check(host, &clusterInfo{slowStart: types.SlowStart{
		Mode:             "not exists",
		Aggression:       v2.SlowStartDefaultAggression,
		MinWeightPercent: v2.SlowStartDefaultMinWeightPercent,
	}}, 100, true)

	f = 1
	check(host, &clusterInfo{slowStart: types.SlowStart{
		Mode:             "test",
		Aggression:       v2.SlowStartDefaultAggression,
		MinWeightPercent: v2.SlowStartDefaultMinWeightPercent,
	}}, 100, false)

	f = 0.01
	check(host, &clusterInfo{slowStart: types.SlowStart{
		Mode:             "test",
		Aggression:       0.1,
		MinWeightPercent: 0.1,
	}}, 10, false)

	f = 10
	check(host, &clusterInfo{slowStart: types.SlowStart{
		Mode:             "test",
		Aggression:       1.0,
		MinWeightPercent: v2.SlowStartDefaultMinWeightPercent,
	}}, 100, false)

	f = 0.1
	check(host, &clusterInfo{slowStart: types.SlowStart{
		Mode:             "test",
		Aggression:       1.0,
		MinWeightPercent: v2.SlowStartDefaultMinWeightPercent,
	}}, 10, false)

	check(host, &clusterInfo{slowStart: types.SlowStart{
		Mode:             "test",
		Aggression:       2.0,
		MinWeightPercent: v2.SlowStartDefaultMinWeightPercent,
	}}, 31.622776601683793, false)

	check(host, &clusterInfo{slowStart: types.SlowStart{
		Mode:             "test",
		Aggression:       3.0,
		MinWeightPercent: v2.SlowStartDefaultMinWeightPercent,
	}}, 46.4158883361278, false)
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

func TestLeastActiveRequestLoadBalancer_ChooseHost(t *testing.T) {
	verify := func(t *testing.T, info types.ClusterInfo, delta float64) {
		bias := 1.0
		if info != nil && info.LbConfig() != nil {
			bias = info.LbConfig().ActiveRequestBias
		}

		hosts := createHostsetWithStats(exampleHostConfigs(), "test")
		i := 0
		hosts.Range(func(host types.Host) bool {
			i++
			h := host.(*mockHost)
			h.w = uint32(i)
			h.HostStats().UpstreamRequestActive.Inc(int64(i))
			return true
		})

		lb := newLeastActiveRequestLoadBalancer(info, hosts)

		expect := make(map[types.Host]float64)
		actual := make(map[types.Host]float64)
		total := 0.0

		hosts.Range(func(h types.Host) bool {
			weight := h.Weight()
			activeRequest := h.HostStats().UpstreamRequestActive.Count()
			expect[h] = float64(weight) / math.Pow(float64(activeRequest+1), bias)
			total += float64(weight) / math.Pow(float64(activeRequest+1), bias)

			return true
		})

		for i := 0.0; i < total*100000; i++ {
			h := lb.ChooseHost(nil)
			actual[h]++
		}

		compareDistribution(t, expect, actual, delta)
	}

	t.Run("no bias", func(t *testing.T) {
		verify(t, nil, 1e-4)
	})
	t.Run("low bias", func(t *testing.T) {
		verify(t, &clusterInfo{
			lbConfig: &v2.LbConfig{
				ActiveRequestBias: 0.5,
			},
		}, 1e-4)
	})
	t.Run("high bias", func(t *testing.T) {
		verify(t, &clusterInfo{
			lbConfig: &v2.LbConfig{
				ActiveRequestBias: 1.5,
			},
		}, 1e-4)
	})
}

func compareDistribution(t *testing.T, expect, actual map[types.Host]float64, delta float64) {
	normalize := func(counter map[types.Host]float64) map[types.Host]float64 {
		total := 0.0
		for _, v := range counter {
			total += v
		}

		result := make(map[types.Host]float64)
		for k, v := range counter {
			result[k] = v / total
		}

		return result
	}

	expect = normalize(expect)
	actual = normalize(actual)

	for h, e := range expect {
		assert.InDelta(t, e, actual[h], delta)
	}
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

func Test_PeakEwmaLoadBalancer(t *testing.T) {
	log.DefaultLogger.SetLogLevel(log.DEBUG)

	now := time.Time{}
	supermonkey.Patch(time.Now, func() time.Time {
		return now
	})

	info := &clusterInfo{connectTimeout: time.Second, idleTimeout: time.Second}

	t.Run("clusterInfo is nil", func(t *testing.T) {
		lb := newPeakEwmaLoadBalancer(nil, NewHostSet(mockHostList(0, "", nil)))
		assert.NotNil(t, lb)
	})

	t.Run("no host", func(t *testing.T) {
		hs := &hostSet{allHosts: mockHostList(0, "", nil)}
		lb := newPeakEwmaLoadBalancer(info, hs)
		assert.False(t, lb.IsExistsHosts(nil))
		assert.Equal(t, 0, lb.HostNum(nil))
		h := lb.ChooseHost(nil)
		assert.Nil(t, h)
	})

	t.Run("only 1 host", func(t *testing.T) {
		hs := &hostSet{allHosts: mockHostList(1, "", nil)}
		lb := newPeakEwmaLoadBalancer(info, hs)
		assert.True(t, lb.IsExistsHosts(nil))
		assert.Equal(t, 1, lb.HostNum(nil))
		h := lb.ChooseHost(nil)
		assert.Equal(t, "0", h.Hostname())
	})

	t.Run("only 1 host unhealthy", func(t *testing.T) {
		hs := &hostSet{allHosts: mockHostList(1, "", nil)}
		hs.Get(0).SetHealthFlag(api.FAILED_ACTIVE_HC)
		lb := newPeakEwmaLoadBalancer(info, hs)
		assert.True(t, lb.IsExistsHosts(nil))
		assert.Equal(t, 1, lb.HostNum(nil))
		h := lb.ChooseHost(nil)
		assert.Nil(t, h)
	})

	t.Run("should choose average shortest in small cluster", func(t *testing.T) {
		info := NewClusterInfo(v2.Cluster{}).(*clusterInfo)
		info.stats = newClusterStats("mock")

		hs := NewHostSet(mockHostList(2, "", info))

		mh := hs.Get(0).(*mockHost)
		mh.w = 1
		mh.stats = newHostStats("mock", mh.addr)
		mh.stats.UpstreamRequestDurationEWMA.Update(1)
		mh.stats.UpstreamRequestDurationEWMA.Update(2)
		info.stats.UpstreamRequestDurationEWMA.Update(1)
		info.stats.UpstreamRequestDurationEWMA.Update(2)
		mh.stats.UpstreamRequestActive.Inc(1)
		mh.stats.UpstreamResponseSuccess.Inc(2)
		mh.stats.UpstreamRequestTotal.Inc(2)

		mh = hs.Get(1).(*mockHost)
		mh.w = 1
		mh.stats = newHostStats("mock", mh.addr)
		mh.stats.UpstreamRequestDurationEWMA.Update(2)
		info.stats.UpstreamRequestDurationEWMA.Update(2)
		mh.stats.UpstreamRequestDurationEWMA.Update(4)
		info.stats.UpstreamRequestDurationEWMA.Update(4)
		mh.stats.UpstreamRequestActive.Inc(1)
		mh.stats.UpstreamResponseSuccess.Inc(2)
		mh.stats.UpstreamRequestTotal.Inc(2)

		// Wait for the EWMA to tick
		now = now.Add(time.Second)

		lb := newPeakEwmaLoadBalancer(info, hs)
		assert.True(t, lb.IsExistsHosts(nil))
		assert.Equal(t, 2, lb.HostNum(nil))
		h := lb.ChooseHost(nil)
		assert.Equal(t, "0", h.Hostname())
	})

	t.Run("should choose average smallest score with biasedActiveRequest in small cluster", func(t *testing.T) {
		info := &clusterInfo{
			lbConfig: &v2.LbConfig{ChoiceCount: 2, ActiveRequestBias: 3.0},
			stats:    newClusterStats("mock"),
		}

		hs := NewHostSet(mockHostList(2, "", info))

		mh := hs.Get(0).(*mockHost)
		mh.w = 1
		mh.stats = newHostStats("mock", mh.addr)
		mh.stats.UpstreamRequestDurationEWMA.Update(1)
		mh.stats.UpstreamRequestDurationEWMA.Update(2)
		info.stats.UpstreamRequestDurationEWMA.Update(1)
		info.stats.UpstreamRequestDurationEWMA.Update(2)
		mh.stats.UpstreamRequestActive.Inc(2)

		mh = hs.Get(1).(*mockHost)
		mh.w = 1
		mh.stats = newHostStats("mock", mh.addr)
		mh.stats.UpstreamRequestDurationEWMA.Update(2)
		mh.stats.UpstreamRequestDurationEWMA.Update(4)
		info.stats.UpstreamRequestDurationEWMA.Update(2)
		info.stats.UpstreamRequestDurationEWMA.Update(4)
		mh.stats.UpstreamRequestActive.Inc(1)

		// Wait for the EWMA to tick
		now = now.Add(time.Second)

		lb := newPeakEwmaLoadBalancer(info, hs)
		assert.True(t, lb.IsExistsHosts(nil))
		assert.Equal(t, 2, lb.HostNum(nil))
		h := lb.ChooseHost(nil)
		assert.Equal(t, "1", h.Hostname())
	})

	t.Run("should choose average not worst in large cluster", func(t *testing.T) {
		info := NewClusterInfo(v2.Cluster{}).(*clusterInfo)
		info.stats = newClusterStats("mock")

		hs := NewHostSet(mockHostList(9, "", info))
		hs.Range(func(host types.Host) bool {
			mh := host.(*mockHost)
			mh.w = uint32(rand.Intn(10) + 1)
			mh.stats = newHostStats("mock", mh.addr)
			for j, rnd := 0, rand.Intn(10)+1; j < rnd; j++ {
				duration := int64(rand.Intn(5) + 1)
				mh.stats.UpstreamRequestDurationEWMA.Update(duration)
				info.stats.UpstreamRequestDurationEWMA.Update(duration)
				mh.stats.UpstreamRequestActive.Inc(int64(rand.Intn(1)))
			}
			return true
		})

		now = now.Add(time.Second)

		lb := newPeakEwmaLoadBalancer(info, hs).(*peakEwmaLoadBalancer)
		assert.True(t, lb.IsExistsHosts(nil))
		assert.Equal(t, 9, lb.HostNum(nil))
		h := lb.ChooseHost(nil)
		worst := true
		hs.Range(func(host types.Host) bool {
			if lb.unweightedPeakEwmaScore(host) > lb.unweightedPeakEwmaScore(h) {
				worst = false
				return false
			}
			return true
		})
		assert.False(t, worst)
	})

	t.Run("fallback if metrics is disabled", func(t *testing.T) {
		info := NewClusterInfo(v2.Cluster{}).(*clusterInfo)
		info.stats = newClusterStats("mock")

		metrics.SetStatsMatcher(true, nil, nil)
		hs := NewHostSet(mockHostList(10, "", info))
		hs.Range(func(host types.Host) bool {
			mh := host.(*mockHost)
			mh.w = 1
			mh.stats = newHostStats("mock", mh.addr)
			return true
		})
		lb := newPeakEwmaLoadBalancer(info, hs)

		hosts := make(map[types.Host]bool)
		for i := 0; i < 100; i++ {
			h := lb.ChooseHost(nil)
			assert.NotNil(t, h)
			hosts[h] = true
		}
		assert.True(t, len(hosts) > 1)
		metrics.SetStatsMatcher(false, nil, nil)
	})

	t.Run("iterateChoose fallback if most hosts are unhealthy", func(t *testing.T) {
		cluster := NewClusterInfo(v2.Cluster{}).(*clusterInfo)
		cluster.stats = newClusterStats("mock")

		hs := NewHostSet(mockHostList(2, "", cluster))
		hs.Range(func(host types.Host) bool {
			mh := host.(*mockHost)
			mh.w = 1
			mh.stats = newHostStats("mock", mh.addr)
			return true
		})
		hs.Get(0).SetHealthFlag(api.FAILED_ACTIVE_HC)

		lb := newPeakEwmaLoadBalancer(info, hs)
		for i := 0; i < 100; i++ {
			h := lb.ChooseHost(nil)
			assert.NotNil(t, h)
		}
	})

	t.Run("randomChoose fallback if most hosts are unhealthy", func(t *testing.T) {
		cluster := NewClusterInfo(v2.Cluster{}).(*clusterInfo)
		cluster.stats = newClusterStats("mock")

		hs := NewHostSet(mockHostList(10, "", cluster))
		hs.Range(func(host types.Host) bool {
			mh := host.(*mockHost)
			mh.w = 1
			mh.stats = newHostStats("mock", mh.addr)
			if rand.Intn(2) == 0 {
				host.SetHealthFlag(api.FAILED_ACTIVE_HC)
			}
			return true
		})
		lb := newPeakEwmaLoadBalancer(info, hs)
		for i := 0; i < 100; i++ {
			h := lb.ChooseHost(nil)
			assert.NotNil(t, h)
		}
	})

	t.Run("ewma is not fixed if no errors", func(t *testing.T) {
		h1 := &mockHost{stats: newHostStats("mock", "127.0.0.1")}
		h1.HostStats().UpstreamRequestDurationEWMA.Update(1)

		h2 := &mockHost{stats: newHostStats("mock", "127.0.0.2")}
		h2.HostStats().UpstreamRequestDurationEWMA.Update(1)

		lb := peakEwmaLoadBalancer{} // just for `unweightedPeakEwmaScore`

		for i := 0; i < 10; i++ {
			now = now.Add(time.Second)
			assert.Equal(t, lb.unweightedPeakEwmaScore(h1), lb.unweightedPeakEwmaScore(h2))
		}
	})

	t.Run("ewma error rate is still decaying", func(t *testing.T) {
		host := &mockHost{stats: newHostStats("mock", "127.0.0.1")}
		host.HostStats().UpstreamRequestDurationEWMA.Update(1)
		host.HostStats().UpstreamResponseTotalEWMA.Update(10)
		host.HostStats().UpstreamResponseClientErrorEWMA.Update(1)
		host.HostStats().UpstreamResponseServerErrorEWMA.Update(1)

		lb := peakEwmaLoadBalancer{} // just for `unweightedPeakEwmaScore`

		prescore := 100.0
		for i := 0; i < 10; i++ {
			now = now.Add(time.Second)
			score := lb.unweightedPeakEwmaScore(host)
			assert.Less(t, score, prescore)
			prescore = score
		}
	})
}

func BenchmarkShortestResponseLoadBalancer_ChooseHost_Weighted(b *testing.B) {
	b.StopTimer()
	info := &clusterInfo{connectTimeout: time.Second, idleTimeout: time.Second}

	hs := NewHostSet(mockHostList(10, "", info))
	hs.Range(func(host types.Host) bool {
		mh := host.(*mockHost)
		mh.w = uint32(rand.Intn(10) + 1)
		mh.stats = newHostStats("mock", mh.addr)
		for i, n := 0, rand.Intn(1000)+1; i < n; i++ {
			mh.stats.UpstreamRequestDurationEWMA.Update(int64(rand.Intn(5)))
			mh.stats.UpstreamRequestActive.Inc(int64(rand.Intn(1)))
			mh.stats.UpstreamResponseSuccess.Inc(int64(rand.Intn(1)))
			mh.stats.UpstreamRequestTotal.Inc(1)
			mh.stats.UpstreamConnectionConFail.Inc(int64(rand.Intn(1)))
			mh.stats.UpstreamConnectionTotal.Inc(1)
		}
		return true
	})

	lb := newPeakEwmaLoadBalancer(info, hs)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		lb.ChooseHost(nil)
	}
	b.StopTimer()
}

func BenchmarkShortestResponseLoadBalancer_ChooseHost_Unweighted(b *testing.B) {
	b.StopTimer()
	info := &clusterInfo{connectTimeout: time.Second, idleTimeout: time.Second}

	hs := NewHostSet(mockHostList(10, "", nil))
	hs.Range(func(host types.Host) bool {
		mh := host.(*mockHost)
		mh.w = 1
		mh.stats = newHostStats("mock", mh.addr)
		for i, n := 0, rand.Intn(1000)+1; i < n; i++ {
			mh.stats.UpstreamRequestDurationEWMA.Update(int64(rand.Intn(5)))
			mh.stats.UpstreamRequestActive.Inc(int64(rand.Intn(1)))
			mh.stats.UpstreamResponseSuccess.Inc(int64(rand.Intn(1)))
			mh.stats.UpstreamRequestTotal.Inc(1)
			mh.stats.UpstreamConnectionConFail.Inc(int64(rand.Intn(1)))
			mh.stats.UpstreamConnectionTotal.Inc(1)
		}
		return true
	})

	lb := newPeakEwmaLoadBalancer(info, hs)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		lb.ChooseHost(nil)
	}
	b.StopTimer()
}
