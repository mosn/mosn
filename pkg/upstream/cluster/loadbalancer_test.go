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
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"mosn.io/api"

	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/types"
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
	lb := newWeightedRRLoadBalancer(nil, hs)
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
	lb := newWeightedRRLoadBalancer(nil, hs)
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
		lb := newWeightedRRLoadBalancer(nil, hs)
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
		lb := newWeightedRRLoadBalancer(nil, hs)
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
