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
	"testing"

	"sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/types"
)

func BenchmarkRandomLB(b *testing.B) {
	hostSet := &hostSet{}
	hosts := makePool(10).MakeHosts(10, nil)
	hostSet.UpdateHosts(hosts)
	lb := newRandomLoadBalancer(hostSet)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lb.ChooseHost(nil)
		}
	})
}

func BenchmarkRoundRobinLB(b *testing.B) {
	hostSet := &hostSet{}
	hosts := makePool(10).MakeHosts(10, nil)
	hostSet.UpdateHosts(hosts)
	lb := newRoundRobinLoadBalancer(hostSet)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lb.ChooseHost(nil)
		}
	})

}

func BenchmarkSubsetLB(b *testing.B) {
	hostSet := &hostSet{}
	subsetConfig := &v2.LBSubsetConfig{
		FallBackPolicy: 2,
		SubsetSelectors: [][]string{
			[]string{
				"zone",
			},
			[]string{
				"zone", "version",
			},
		},
	}
	pool := makePool(10)
	var hosts []types.Host
	hosts = append(hosts, pool.MakeHosts(5, v2.Metadata{
		"zone":    "RZ41A",
		"version": "1.0.0",
	})...)
	hosts = append(hosts, pool.MakeHosts(5, v2.Metadata{
		"zone":    "RZ41A",
		"version": "2.0.0",
	})...)
	hostSet.UpdateHosts(hosts)
	lb := NewSubsetLoadBalancer(types.Random, hostSet, newClusterStats("BenchmarkSubsetLB"), NewLBSubsetInfo(subsetConfig))
	b.Run("CtxZone", func(b *testing.B) {
		ctx := newMockLbContext(map[string]string{
			"zone": "RZ41A",
		})
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				lb.ChooseHost(ctx)
			}
		})
	})
	b.Run("CtxZoneAndVersion", func(b *testing.B) {
		ctx := newMockLbContext(map[string]string{
			"zone":    "RZ41A",
			"version": "1.0.0",
		})
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				lb.ChooseHost(ctx)
			}
		})
	})
	b.Run("CtxNotMatched", func(b *testing.B) {
		ctx := newMockLbContext(map[string]string{
			"version": "1.0.0",
		})
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				lb.ChooseHost(ctx)
			}
		})
	})
}
