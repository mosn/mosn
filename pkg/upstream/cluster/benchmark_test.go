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
	"math/rand"
	"os"
	"testing"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

func TestMain(m *testing.M) {
	log.DefaultLogger.SetLogLevel(log.ERROR)
	os.Exit(m.Run())
}

func BenchmarkHostConfig(b *testing.B) {
	host := &simpleHost{
		hostname:      "Testhost",
		addressString: "127.0.0.1:8080",
		weight:        100,
		metaData: api.Metadata{
			"zone":    "a",
			"version": "1",
		},
	}
	b.Run("Host.Config", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			host.Config()
		}
	})
}

func BenchmarkAddOrUpdateCluster(b *testing.B) {
	_createClusterManager()
	adapter := GetClusterMngAdapterInstance()
	// host count: 100, 500, 1000, 5000, 20000
	for _, count := range []int{100, 500, 1000, 5000, 20000} {
		pool := makePool(count)
		hosts := make([]v2.Host, 0, count)
		for i := 0; i < count; i++ {
			h := v2.Host{
				HostConfig: v2.HostConfig{
					Address: pool.Get(),
				},
			}
			hosts = append(hosts, h)
		}
		if err := adapter.UpdateClusterHosts("test1", hosts); err != nil {
			b.Fatal("prepare cluster failed")
		}
		c := v2.Cluster{
			Name:   "test1",
			LbType: v2.LB_RANDOM,
			TLS: v2.TLSConfig{
				Status:       true,
				InsecureSkip: true,
			},
		}
		b.Run(fmt.Sprintf("update_count:%d", count), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				adapter.AddOrUpdatePrimaryCluster(c)
			}
		})
	}
}

func BenchmarkUpdateClusterHosts(b *testing.B) {
	cluster := _createTestCluster()
	// assume cluster have 1000 hosts
	pool := makePool(1010)
	hosts := make([]types.Host, 0, 1000)
	metas := []api.Metadata{
		api.Metadata{"version": "1", "zone": "a"},
		api.Metadata{"version": "1", "zone": "b"},
		api.Metadata{"version": "2", "zone": "a"},
	}
	for _, meta := range metas {
		hosts = append(hosts, pool.MakeHosts(300, meta)...)
	}
	noLabelHosts := pool.MakeHosts(100, nil)
	hosts = append(hosts, noLabelHosts...)
	// hosts changes, some are removed, some are added
	var newHosts []types.Host
	for idx := range metas {
		newHosts = append(newHosts, hosts[idx:idx*300+5]...)
	}
	newHosts = append(newHosts, pool.MakeHosts(10, api.Metadata{
		"version": "3",
		"zone":    "b",
	})...)
	newHosts = append(newHosts, noLabelHosts...)
	b.Run("UpdateClusterHost", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cluster.UpdateHosts(NewHostSet(hosts))
			b.StartTimer()
			cluster.UpdateHosts(NewHostSet(newHosts))
		}
	})

}

func BencmarkUpdateClusterHostsLabel(b *testing.B) {
	cluster := _createTestCluster()
	pool := makePool(1000)
	hosts := make([]types.Host, 0, 1000)
	metas := []api.Metadata{
		api.Metadata{"zone": "a"},
		api.Metadata{"zone": "b"},
	}
	for _, meta := range metas {
		hosts = append(hosts, pool.MakeHosts(500, meta)...)
	}
	newHosts := make([]types.Host, 1000)
	copy(newHosts, hosts)
	for i := 0; i < 500; i++ {
		host := newHosts[i]
		var ver string
		if i%2 == 0 {
			ver = "1.0"
		} else {
			ver = "2.0"
		}
		newHost := &mockHost{
			addr: host.AddressString(),
			meta: api.Metadata{
				"version": ver,
				"zone":    "a",
			},
		}
		newHosts[i] = newHost
	}
	b.Run("UpdateClusterHostsLabel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cluster.UpdateHosts(NewHostSet(hosts))
			b.StartTimer()
			cluster.UpdateHosts(NewHostSet(newHosts))
		}
	})
}

func BenchmarkClusterManagerRemoveHosts(b *testing.B) {
	_createClusterManager()
	pool := makePool(1200)
	hostsConfig := make([]v2.Host, 0, 1200)
	metas := []api.Metadata{
		api.Metadata{"version": "1", "zone": "a"},
		api.Metadata{"version": "1", "zone": "b"},
		api.Metadata{"version": "2", "zone": "a"},
		nil,
	}
	for _, meta := range metas {
		hosts := pool.MakeHosts(300, meta)
		for _, host := range hosts {
			hostsConfig = append(hostsConfig, v2.Host{
				HostConfig: v2.HostConfig{
					Address: host.AddressString(),
				},
				MetaData: meta,
			})
		}
	}
	var removeList []string
	for i := 0; i < 4; i++ {
		for j := 0; j < 10; j++ {
			host := hostsConfig[i*300+j]
			removeList = append(removeList, host.Address)
		}
	}
	b.Run("ClusterRemoveHosts", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			GetClusterMngAdapterInstance().UpdateClusterHosts("test1", hostsConfig)
			b.StartTimer()
			GetClusterMngAdapterInstance().RemoveClusterHosts("test1", removeList)
		}
	})

}

func BenchmarkClusterManagerAppendHosts(b *testing.B) {
	_createClusterManager()
	pool := makePool(1240)
	hostsConfig := make([]v2.Host, 0, 1200)
	metas := []api.Metadata{
		api.Metadata{"version": "1", "zone": "a"},
		api.Metadata{"version": "1", "zone": "b"},
		api.Metadata{"version": "2", "zone": "a"},
		nil,
	}
	for _, meta := range metas {
		hosts := pool.MakeHosts(300, meta)
		for _, host := range hosts {
			hostsConfig = append(hostsConfig, v2.Host{
				HostConfig: v2.HostConfig{
					Address: host.AddressString(),
				},
				MetaData: meta,
			})
		}
	}
	var addHostsCfg []v2.Host
	for _, meta := range metas {
		hs := pool.MakeHosts(10, nil)
		for _, h := range hs {
			addHostsCfg = append(addHostsCfg, v2.Host{
				HostConfig: v2.HostConfig{
					Address: h.AddressString(),
				},
				MetaData: meta,
			})
		}
	}
	b.Run("ClusterManagerAppendHosts", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			GetClusterMngAdapterInstance().UpdateClusterHosts("test1", hostsConfig)
			b.StartTimer()
			GetClusterMngAdapterInstance().AppendClusterHosts("test1", addHostsCfg)
		}
	})
}

func BenchmarkRandomLB(b *testing.B) {
	hostSet := &hostSet{}
	hosts := makePool(10).MakeHosts(10, nil)
	hostSet.setFinalHost(hosts)
	lb := newRandomLoadBalancer(nil, hostSet)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lb.ChooseHost(nil)
		}
	})
}

func BenchmarkRandomLBWithUnhealthyHost(b *testing.B) {
	hostSet := &hostSet{}
	hosts := makePool(10).MakeHosts(10, nil)
	hostSet.setFinalHost(hosts)
	lb := newRandomLoadBalancer(nil, hostSet)
	for i := 0; i < 5; i++ {
		hosts[i].SetHealthFlag(api.FAILED_ACTIVE_HC)
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lb.ChooseHost(nil)
		}
	})
}

func BenchmarkRoundRobinLB(b *testing.B) {
	hostSet := &hostSet{}
	hosts := makePool(10).MakeHosts(10, nil)
	hostSet.setFinalHost(hosts)
	lb := rrFactory.newRoundRobinLoadBalancer(nil, hostSet)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lb.ChooseHost(nil)
		}
	})
}

func BenchmarkRoundRobinLBWithUnhealthyHost(b *testing.B) {
	hostSet := &hostSet{}
	hosts := makePool(10).MakeHosts(10, nil)
	hostSet.setFinalHost(hosts)
	lb := rrFactory.newRoundRobinLoadBalancer(nil, hostSet)
	for i := 0; i < 5; i++ {
		hosts[i].SetHealthFlag(api.FAILED_OUTLIER_CHECK)
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lb.ChooseHost(nil)
		}
	})
}

func BenchmarkLeastActiveRequestLB(b *testing.B) {
	hostSet := &hostSet{}
	hosts := makePool(10).MakeHosts(10, map[string]string{"cluster": ""})
	hostSet.setFinalHost(hosts)
	lb := newLeastActiveRequestLoadBalancer(nil, hostSet)
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
	hosts = append(hosts, pool.MakeHosts(5, api.Metadata{
		"zone":    "RZ41A",
		"version": "1.0.0",
	})...)
	hosts = append(hosts, pool.MakeHosts(5, api.Metadata{
		"zone":    "RZ41A",
		"version": "2.0.0",
	})...)
	hostSet.setFinalHost(hosts)
	lb := newSubsetLoadBalancer(types.Random, hostSet, newClusterStats("BenchmarkSubsetLB"), NewLBSubsetInfo(subsetConfig))
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

func BenchmarkMaglevLB(b *testing.B) {
	hostSet := getMockHostSet(20000)
	mgvLb := newMaglevLoadBalancer(nil, hostSet)

	testProtocol := types.ProtocolName("SomeProtocol")
	mockRoute := &mockRoute{
		routeRule: &mockRouteRule{
			policy: &mockPolicy{
				hashPolicy: &mockHashPolicy{},
			},
		},
	}
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, testProtocol)
	lbctx := &mockLbContext{
		context: ctx,
		route:   mockRoute,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = mgvLb.ChooseHost(lbctx)
	}
}

func BenchmarkMaglevLBParallel(b *testing.B) {
	hostSet := getMockHostSet(20000)
	mgvLb := newMaglevLoadBalancer(nil, hostSet)

	testProtocol := types.ProtocolName("SomeProtocol")
	mockRoute := &mockRoute{
		routeRule: &mockRouteRule{
			policy: &mockPolicy{
				hashPolicy: &mockHashPolicy{},
			},
		},
	}
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, testProtocol)
	lbctx := &mockLbContext{
		context: ctx,
		route:   mockRoute,
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = mgvLb.ChooseHost(lbctx)
		}
	})
}

func BenchmarkMaglevLBFallback(b *testing.B) {
	hostSet := getMockHostSet(20000)
	mgvLb := newMaglevLoadBalancer(nil, hostSet)

	// make sure 0 index hasn -> host-15748 is unhealthy, to ensure fallback
	hostSet.Hosts()[15748].SetHealthFlag(api.FAILED_ACTIVE_HC)
	// randomly set 10000 of 20000 host unhealthy
	rand.Seed(0)
	for i := 0; i < 10000; i++ {
		randIndex := rand.Intn(19999)
		hostSet.Hosts()[randIndex].SetHealthFlag(api.FAILED_ACTIVE_HC)
	}

	testProtocol := types.ProtocolName("SomeProtocol")
	mockRoute := &mockRoute{
		routeRule: &mockRouteRule{
			policy: &mockPolicy{
				hashPolicy: &mockHashPolicy{},
			},
		},
	}
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, testProtocol)
	lbctx := &mockLbContext{
		context: ctx,
		route:   mockRoute,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = mgvLb.ChooseHost(lbctx)
	}
}
