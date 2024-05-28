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
	"testing"

	"github.com/stretchr/testify/assert"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

func TestLACChooseHost(t *testing.T) {
	hosts := createHostsetWithStats(exampleHostConfigs(), "test")
	balancer := NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveConnection}, hosts)
	host := balancer.ChooseHost(newMockLbContext(nil))
	assert.NotNil(t, host)

	hosts.Range(func(host types.Host) bool {
		mockRequest(host, true, 10)
		return true
	})
	// new lb to refresh edf
	balancer = NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveConnection}, hosts)
	actual := balancer.ChooseHost(newMockLbContext(nil))
	assert.NotNil(t, actual)

	// test only one host
	h := exampleHostConfigs()[0:1]
	hosts = createHostsetWithStats(h, "test")
	balancer = NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveConnection}, hosts)
	actual = balancer.ChooseHost(nil)
	assert.Equal(t, hosts.allHosts[0], actual)

	// test no host
	h = exampleHostConfigs()[0:0]
	hosts = createHostsetWithStats(h, "test")
	balancer = NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveConnection}, hosts)
	actual = balancer.ChooseHost(nil)
	assert.Nil(t, actual)
}

func TestLeastActiveConnectionLoadBalancer_ChooseHost(t *testing.T) {
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
			h.HostStats().UpstreamConnectionActive.Inc(int64(i))
			return true
		})

		lb := newLeastActiveConnectionLoadBalancer(info, hosts)

		expect := make(map[types.Host]float64)
		actual := make(map[types.Host]float64)

		hosts.Range(func(h types.Host) bool {
			weight := h.Weight()
			activeConnection := h.HostStats().UpstreamConnectionActive.Count()
			expect[h] = float64(weight) / math.Pow(float64(activeConnection+1), bias)

			return true
		})

		for i := 0.0; i < 100000; i++ {
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
