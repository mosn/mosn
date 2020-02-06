package cluster

import (
	"testing"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

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

func TestClusterUpdateHosts(t *testing.T) {
	cluster := _createTestCluster()
	// init hosts
	pool := makePool(100)
	var hosts []types.Host
	metas := []api.Metadata{
		api.Metadata{"version": "1", "zone": "a"},
		api.Metadata{"version": "1", "zone": "b"},
		api.Metadata{"version": "2", "zone": "a"},
		nil, // no meta (in any point)
	}
	for _, meta := range metas {
		hosts = append(hosts, pool.MakeHosts(10, meta)...)
	}
	cluster.UpdateHosts(hosts)
	// verify
	snap := cluster.Snapshot()
	subLb := snap.LoadBalancer().(*subsetLoadBalancer)
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
	newHosts = append(newHosts, pool.MakeHosts(10, api.Metadata{
		"version": "2",
		"zone":    "b",
	})...)
	newHosts = append(newHosts, pool.MakeHosts(10, api.Metadata{
		"version": "3",
		"ignore":  "true",
	})...)
	cluster.UpdateHosts(newHosts)
	newSnap := cluster.Snapshot()
	newSubLb := newSnap.LoadBalancer().(*subsetLoadBalancer)
	// verify
	if len(newSubLb.fallbackSubset.hostSet.Hosts()) != 60 {
		t.Fatal("default fallback should be all hosts")
	}
	newResult := &subSetMapResult{
		result: map[string][]string{},
	}
	newResult.RangeSubsetMap("", newSubLb.subSets)
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

func TestUpdateHostLabels(t *testing.T) {
	cluster := _createTestCluster()
	host := &mockHost{
		addr: "127.0.0.1:8080",
		meta: api.Metadata{
			"version": "1",
		},
	}
	cluster.UpdateHosts([]types.Host{host})
	snap := cluster.Snapshot()
	subLb := snap.LoadBalancer().(*subsetLoadBalancer)
	result := &subSetMapResult{
		result: map[string][]string{},
	}
	result.RangeSubsetMap("", subLb.subSets)
	expectedResult := map[string]int{
		"version->1->": 1,
	}
	if len(result.result) != 1 {
		t.Fatalf("expected 1 subsets, but got: %d", len(result.result))
	}
	for p, count := range expectedResult {
		sub, ok := result.result[p]
		if !ok || len(sub) != count {
			t.Fatalf("%s is not expected, exists: %v, count: %d", p, ok, len(sub))
		}
	}
	// update host label
	newHost := &mockHost{
		addr: "127.0.0.1:8080",
		meta: api.Metadata{
			"zone":    "a",
			"version": "2",
		},
	}
	cluster.UpdateHosts([]types.Host{newHost})
	newSnap := cluster.Snapshot()
	newSubLb := newSnap.LoadBalancer().(*subsetLoadBalancer)
	newResult := &subSetMapResult{
		result: map[string][]string{},
	}
	newResult.RangeSubsetMap("", newSubLb.subSets)
	newExpectedResult := map[string]int{
		"version->2->":          1,
		"version->2->zone->a->": 1,
	}
	if len(newResult.result) != 2 {
		t.Fatalf("expected 2 subsets, but got: %d", len(newResult.result))
	}
	for p, count := range newExpectedResult {
		sub, ok := newResult.result[p]
		if !ok || len(sub) != count {
			t.Fatalf("%s is not expected, exists: %v, count: %d", p, ok, len(sub))
		}
	}
}
