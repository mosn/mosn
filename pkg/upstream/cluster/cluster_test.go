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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
				{"version"},
				{"version", "zone"},
			},
		},
	}
	return NewCluster(clusterConfig)
}

func TestNewClusterInfoWithDefaultSlowStartConfig(t *testing.T) {
	clusterConfig := v2.Cluster{
		Name:      "test_cluster",
		LbType:    v2.LB_ROUNDROBIN,
		SlowStart: v2.SlowStartConfig{},
	}
	clusterInfo := NewClusterInfo(clusterConfig)
	assert.Zero(t, clusterInfo.SlowStart().SlowStartDuration)
	assert.Equal(t, 1.0, clusterInfo.SlowStart().Aggression)        // 1.0 by default
	assert.Equal(t, 0.10, clusterInfo.SlowStart().MinWeightPercent) // 10% by default
}

func TestClusterUpdateHosts(t *testing.T) {
	cluster := _createTestCluster()
	// init hosts
	pool := makePool(100)
	var hosts []types.Host
	metas := []api.Metadata{
		{"version": "1", "zone": "a"},
		{"version": "1", "zone": "b"},
		{"version": "2", "zone": "a"},
		nil, // no meta (in any point)
	}
	for _, meta := range metas {
		hosts = append(hosts, pool.MakeHosts(10, meta)...)
	}
	cluster.UpdateHosts(NewHostSet(hosts))
	// verify
	snap := cluster.Snapshot()
	subLb := snap.LoadBalancer().(*subsetLoadBalancer)
	if subLb.fallbackSubset.HostNum() != 40 {
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
	cluster.UpdateHosts(NewHostSet(newHosts))
	newSnap := cluster.Snapshot()
	newSubLb := newSnap.LoadBalancer().(*subsetLoadBalancer)
	// verify
	if newSubLb.fallbackSubset.HostNum() != 60 {
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
	cluster.UpdateHosts(NewHostSet([]types.Host{host}))
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
	cluster.UpdateHosts(NewHostSet([]types.Host{newHost}))
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

func TestClusterUseClusterManagerTLS(t *testing.T) {
	clusterManagerInstance.Destroy()
	NewClusterManagerSingleton(nil, nil, nil)
	clusterConfig := v2.Cluster{
		Name:              "test_cluster",
		LbType:            v2.LB_RANDOM,
		ClusterManagerTLS: true,
	}
	c := NewCluster(clusterConfig)
	snap := c.Snapshot()
	if mng := snap.ClusterInfo().TLSMng(); mng.Enabled() {
		t.Fatal("tls should not enabled")
	}
	// update tls mananger
	clusterManagerInstance.UpdateTLSManager(&v2.TLSConfig{
		Status:       true,
		InsecureSkip: true,
	})
	if mng := snap.ClusterInfo().TLSMng(); !mng.Enabled() {
		t.Fatal("tls should enabled")
	}
	// config without manager tls
	clusterConfig2 := v2.Cluster{
		Name:   "test_cluster",
		LbType: v2.LB_RANDOM,
	}
	c2 := NewCluster(clusterConfig2)
	if mng := c2.Snapshot().ClusterInfo().TLSMng(); mng.Enabled() {
		t.Fatal("tls should not enabled")
	}

}

func TestClusterTypeCompatible(t *testing.T) {
	// keep static/dynamic/eds as simple
	for _, typ := range []v2.ClusterType{
		v2.EDS_CLUSTER,
		v2.STATIC_CLUSTER,
		v2.DYNAMIC_CLUSTER,
	} {
		cfg := v2.Cluster{
			Name:        "test",
			ClusterType: typ,
		}
		c := NewCluster(cfg)
		require.Equal(t, v2.SIMPLE_CLUSTER, c.Snapshot().ClusterInfo().ClusterType())
	}
}
