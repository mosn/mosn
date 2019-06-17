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
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/types"
)

func TestMain(m *testing.M) {
	log.DefaultLogger.SetLogLevel(log.ERROR)
	m.Run()
}

func _createTestCluster() types.Cluster {
	clusterConfig := v2.Cluster{
		Name:   "test_cluster",
		LbType: v2.LB_RANDOM,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy: 1, // AnyEndPoint
			SubsetSelectors: [][]string{
				[]string{"version"},
				[]string{"version", "zone"},
			},
		},
	}
	return NewCluster(clusterConfig)
}

func TestClusterUpdateHost(t *testing.T) {
	cluster := _createTestCluster()
	// init host
	pool := makePool(100)
	var hosts []types.Host
	// version 1 zone a
	// version 1 zone b
	// version 2 zone a
	// no meta (in any point)
	metas := []v2.Metadata{
		v2.Metadata{"version": "1", "zone": "a"},
		v2.Metadata{"version": "1", "zone": "b"},
		v2.Metadata{"version": "2", "zone": "a"},
		nil,
	}
	for _, meta := range metas {
		hosts = append(hosts, pool.MakeHosts(10, meta)...)
	}
	cluster.UpdateHosts(hosts)
	// verify
	subLb := cluster.LBInstance().(*subsetLoadBalancer)
	if len(subLb.fallbackSubset.hostSet.Hosts()) != 40 {
		t.Fatal("default fallback should be all hosts")
	}
	result := &subSetMapResult{
		result: map[string][]string{},
	}
	result.RangeSubsetMap("", subLb.subSets)
	if len(result.result) != 5 {
		t.Fatalf("expected 5 subsets, but got: %d", len(result.result))
	}
	for p, sub := range result.result {
		if p == "version->1->" {
			if len(sub) != 20 {
				t.Fatalf("%s host only have %d, expected 20", p, len(sub))
			}
		} else {
			if len(sub) != 10 {
				t.Fatalf("%s host only have %d, expected 10", p, len(sub))
			}
		}
	}
	// create new subset
	// update old subset
	var newHosts []types.Host
	newHosts = append(newHosts, hosts[:5]...)
	newHosts = append(newHosts, hosts[10:15]...)
	newHosts = append(newHosts, hosts[20:]...)
	newHosts = append(newHosts, pool.MakeHosts(10, metas[2])...)
	newHosts = append(newHosts, pool.MakeHosts(10, v2.Metadata{
		"version": "2",
		"zone":    "b",
	})...)
	newHosts = append(newHosts, pool.MakeHosts(10, v2.Metadata{
		"version": "3",
		"ignore":  "true",
	})...)
	cluster.UpdateHosts(newHosts)
	// verify
	if len(subLb.fallbackSubset.hostSet.Hosts()) != 60 {
		t.Fatal("default fallback should be all hosts")
	}
	newResult := &subSetMapResult{
		result: map[string][]string{},
	}
	newResult.RangeSubsetMap("", subLb.subSets)
	if len(newResult.result) != 7 {
		t.Fatalf("expected 7 subsets, but got: %d", len(newResult.result))
	}
	expectedResult := map[string]int{
		"version->1->":          10,
		"version->2->":          30,
		"version->3->":          10,
		"version->1->zone->a->": 5,
		"version->1->zone->b->": 5,
		"version->2->zone->a->": 20,
		"version->2->zone->b->": 10,
	}
	for p, count := range expectedResult {
		sub, ok := newResult.result[p]
		if !ok || len(sub) != count {
			t.Fatalf("%s is not expected, exists: %v, count: %d", p, ok, len(sub))
		}
	}
}

func BenchmarkUpdateHost(b *testing.B) {
	cluster := _createTestCluster()
	// assume cluster have 1000 hosts
	pool := makePool(1010)
	hosts := make([]types.Host, 0, 1000)
	metas := []v2.Metadata{
		v2.Metadata{"version": "1", "zone": "a"},
		v2.Metadata{"version": "1", "zone": "b"},
		v2.Metadata{"version": "2", "zone": "a"},
	}
	for _, meta := range metas {
		hosts = append(hosts, pool.MakeHosts(300, meta)...)
	}
	hosts = append(hosts, pool.MakeHosts(100, nil)...)
	cluster.UpdateHosts(hosts)
	// hosts changes, some are removed, some are added
	var newHosts []types.Host
	for idx := range metas {
		newHosts = append(newHosts, hosts[idx:idx*300+5]...)
	}
	newHosts = append(newHosts, pool.MakeHosts(10, v2.Metadata{
		"version": "3",
		"zone":    "b",
	})...)
	b.Run("UpdateClusterHost", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cluster.UpdateHosts(newHosts)
			cluster.UpdateHosts(hosts)
		}
	})
}
