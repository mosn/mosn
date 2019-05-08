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
	"reflect"
	"sort"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// we test following cases form envoy's example
// see https://github.com/envoyproxy/envoy/blob/master/source/docs/subset_load_balancer.md
// Host List
/*
| host	| stage	| version |	type	 | xlarge
| e1	| prod	| 1.0	  |	std	 | true
| e2	| prod	| 1.0	  |	std
| e3	| prod	| 1.1	  |	std
| e4	| prod	| 1.1	  |	std
| e5	| prod	| 1.0 	  |	bigmem
| e6	| prod	| 1.1	  |	bigmem
| e7	| dev	| 1.2-pre |	std
*/
func ExampleHostConfigs() (hosts []v2.Host) {
	metaDatas := []map[string]string{
		map[string]string{
			"stage":   "prod",
			"version": "1.0",
			"type":    "std",
			"xlarge":  "true", // value should be string, not a bool
		},
		map[string]string{
			"stage":   "prod",
			"version": "1.0",
			"type":    "std",
		},
		map[string]string{
			"stage":   "prod",
			"version": "1.1",
			"type":    "std",
		},
		map[string]string{
			"stage":   "prod",
			"version": "1.1",
			"type":    "std",
		},
		map[string]string{
			"stage":   "prod",
			"version": "1.0",
			"type":    "bigmem",
		},
		map[string]string{
			"stage":   "prod",
			"version": "1.1",
			"type":    "bigmem",
		},
		map[string]string{
			"stage":   "dev",
			"version": "1.2-pre",
			"type":    "std",
		},
	}
	for i, md := range metaDatas {
		hostName := fmt.Sprintf("e%d", i+1)
		addr := fmt.Sprintf("127.0.0.1:808%d", i+1)
		cfg := v2.Host{
			HostConfig: v2.HostConfig{
				Hostname: hostName,
				Address:  addr,
			},
			MetaData: md,
		}
		hosts = append(hosts, cfg)
	}
	return
}

func createPrioritySet(cfg []v2.Host) *prioritySet {
	var hosts []types.Host
	for _, h := range cfg {
		hostInfo := newHostInfo(nil, h, nil)
		host := &host{
			hostInfo: hostInfo,
		}
		hosts = append(hosts, host)
	}
	ps := &prioritySet{}
	ps.GetOrCreateHostSet(0).UpdateHosts(hosts, hosts, hosts, nil)
	return ps
}

// Selector Config
/*
"subset_selectors": [
      { "keys": [ "stage", "type" ] },
      { "keys": [ "stage", "version" ] },
      { "keys": [ "version" ] },
      { "keys": [ "xlarge", "version" ] },
    ]
*/
// mosn's config format is different from envoy
func ExampleSubsetConfig() *v2.LBSubsetConfig {
	return &v2.LBSubsetConfig{
		SubsetSelectors: [][]string{
			[]string{
				"stage", "type",
			},
			[]string{
				"stage", "version",
			},
			[]string{
				"version",
			},
			[]string{
				"xlarge", "version",
			},
		},
	}
}

// LbSubsetMap Result, show as string chain
// stage->dev->type->std->[e7]
// stage->dev->version->1.2-pre->[e7]
// stage->prod->type->std->[e1,e2,e3,e4]
// stage->prod->type->bigmem->[e5,e6]
// stage->prod->version->1.0->[e1,e2,e5]
// stage->prod->version->1.1->[e3,e4,e6]
// version->1.0->[e1,e2,e5]
// version->1.0->xlarge->true->[e1]
// version->1.1->[e3,e4,e6]
// version->1.2-pre->[e7]
var ExampleResult = map[string][]string{
	"stage->dev->type->std->":        []string{"e7"},
	"stage->dev->version->1.2-pre->": []string{"e7"},
	"stage->prod->type->std->":       []string{"e1", "e2", "e3", "e4"},
	"stage->prod->type->bigmem->":    []string{"e5", "e6"},
	"stage->prod->version->1.0->":    []string{"e1", "e2", "e5"},
	"stage->prod->version->1.1->":    []string{"e3", "e4", "e6"},
	"version->1.0->":                 []string{"e1", "e2", "e5"},
	"version->1.0->xlarge->true->":   []string{"e1"},
	"version->1.1->":                 []string{"e3", "e4", "e6"},
	"version->1.2-pre->":             []string{"e7"},
}

func resultMapEqual(m1, m2 map[string][]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for key, node := range m1 {
		node2, ok := m2[key]
		if !ok {
			return false
		}
		if !strSliceEqual(node, node2) {
			return false
		}
	}
	return true
}
func strSliceEqual(s1, s2 []string) bool {
	sort.Strings(s1)
	sort.Strings(s2)
	return reflect.DeepEqual(s1, s2)
}
func strInSlice(s string, list []string) bool {
	for _, ls := range list {
		if s == ls {
			return true
		}
	}
	return false
}

type subSetMapResult struct {
	// key is prefix like: stage->dev->type->std->
	// value is hostname's string like: [e7]
	result map[string][]string
}

// parse a LbSubsetMap as a readable map
func (r *subSetMapResult) RangeSubsetMap(prefix string, subsetMap types.LbSubsetMap) {
	for key, value := range subsetMap {
		for v, entry := range value {
			p := prefix + key + "->" + string(v) + "->"
			if entry.Children() != nil {
				r.RangeSubsetMap(p, entry.Children())
			}
			if entry.PrioritySubset() != nil {
				ps := entry.PrioritySubset().(*PrioritySubsetImpl)
				hosts := ps.prioritySubset.GetHostsInfo(0)
				hostsNode := []string{}
				for _, h := range hosts {
					hostsNode = append(hostsNode, h.Hostname())
				}
				r.result[p] = hostsNode
			}
		}
	}
}

// create a subset as expected, see example
func TestNewSubsetLoadBalancer(t *testing.T) {
	ps := createPrioritySet(ExampleHostConfigs())
	lb := NewSubsetLoadBalancer(types.RoundRobin, ps, newClusterStats("TestNewSubsetLoadBalancer"), NewLBSubsetInfo(ExampleSubsetConfig()))
	if lb == nil {
		t.Fatal("create subset lb failed")
	}
	subSet := lb.(*subSetLoadBalancer).subSets
	result := &subSetMapResult{
		result: map[string][]string{},
	}
	result.RangeSubsetMap("", subSet)
	if !resultMapEqual(result.result, ExampleResult) {
		t.Errorf("subset tree created is not expected, %v", result.result)
	}
}

// MetadataMatchCriteria should exactly matches subset
// case1: stage:prod, version:1.0 shoud find e1,e2,e5
// case2: stage:prod: should find nil
func TestNewSubsetChooseHost(t *testing.T) {
	ps := createPrioritySet(ExampleHostConfigs())
	lb := NewSubsetLoadBalancer(types.RoundRobin, ps, newClusterStats("TestNewSubsetChooseHost"), NewLBSubsetInfo(ExampleSubsetConfig()))
	if lb == nil {
		t.Fatal("create subset lb failed")
	}
	ctx1 := newMockLbContext(map[string]string{
		"stage":   "prod",
		"version": "1.0",
	})
	ctx2 := newMockLbContext(map[string]string{
		"stage": "prod",
	})
	h, ok := lb.TryChooseHostFromContext(ctx1)
	if !ok || h == nil {
		t.Fatal("choose host failed, expected success")
	}
	switch h.Hostname() {
	case "e1", "e2", "e5":
	default:
		t.Fatal("host found, but not the expected subset", h.Hostname())
	}
	if h, ok := lb.TryChooseHostFromContext(ctx2); ok || h != nil {
		t.Fatalf("expected choose failed, but returns a host, host: %v, ok: %v", h, ok)
	}
}

// If selectors not configured the host label, the host will not be put in any subset
func TestNoSubsetHost(t *testing.T) {
	ps := createPrioritySet(ExampleHostConfigs())
	cfg := &v2.LBSubsetConfig{
		SubsetSelectors: [][]string{
			[]string{
				"xlarge", "version",
			},
		},
	}
	// only one host will put in subset (e1)
	// others cannot be found in subset even if version is matched
	lb := NewSubsetLoadBalancer(types.RoundRobin, ps, newClusterStats("TestNoSubsetHost"), NewLBSubsetInfo(cfg))
	if lb == nil {
		t.Fatal("create subset lb failed")
	}
	// found no host
	ctx1 := newMockLbContext(map[string]string{
		"version": "1.0",
	})
	// found host
	ctx2 := newMockLbContext(map[string]string{
		"version": "1.0",
		"xlarge":  "true",
	})
	if h, ok := lb.TryChooseHostFromContext(ctx1); ok || h != nil {
		t.Fatalf("expected choose failed, but returns a host, host: %v, ok: %v", h, ok)
	}
	for i := 0; i < 10; i++ {
		h, ok := lb.TryChooseHostFromContext(ctx2)
		if !ok || h == nil {
			t.Fatal("choose host failed, expected success")
		}
		if h.Hostname() != "e1" {
			t.Fatalf("host found not expected, got: %s", h.Hostname())
		}
	}

}

// TestFallbackWithDefaultSubset configure default subset as fallback
// if a ctx is not matched the subset, use the fallback instead
func TestFallbackWithDefaultSubset(t *testing.T) {
	ps := createPrioritySet(ExampleHostConfigs())
	// only create subset with version and xlarge
	// if not matched, use default subset: stage:dev
	cfg := &v2.LBSubsetConfig{
		FallBackPolicy: uint8(types.DefaultSubsetDefaultSubset),
		DefaultSubset: map[string]string{
			"stage": "dev", // only contain e7
		},
		SubsetSelectors: [][]string{
			[]string{
				"version", "xlarge",
			},
		},
	}
	// ctx1: version:1.0, xlarge: true. match the selector, find e1
	// ctx2: version:1.0, xlarge: false. not matched, find is fallback, e7
	// ctx3: version:1.2, xlarge: true. not matched, find is fallabck, e7
	// ctx4: stage: prod. not matched, find is fallback, e7
	// ctx5~7: nil(mmc is nil/no value). not matched, find is fallback e7
	lb := NewSubsetLoadBalancer(types.RoundRobin, ps, newClusterStats("TestFallbackWithDefaultSubset"), NewLBSubsetInfo(cfg))
	if lb == nil {
		t.Fatal("create subset lb failed")
	}
	testCases := []struct {
		ctx          types.LoadBalancerContext
		expectedHost string
	}{
		{
			ctx: newMockLbContext(map[string]string{
				"version": "1.0",
				"xlarge":  "true",
			}),
			expectedHost: "e1",
		},
		{
			ctx: newMockLbContext(map[string]string{
				"version": "1.0",
				"xlarge":  "false",
			}),
			expectedHost: "e7",
		},
		{
			ctx: newMockLbContext(map[string]string{
				"version": "1.2",
				"xlarge":  "true",
			}),
			expectedHost: "e7",
		},
		{
			ctx: newMockLbContext(map[string]string{
				"stage": "prod",
			}),
			expectedHost: "e7",
		},
		{
			ctx:          nil,
			expectedHost: "e7",
		},
		{
			ctx:          newMockLbContext(nil),
			expectedHost: "e7",
		},
		{
			ctx:          newMockLbContext(map[string]string{}),
			expectedHost: "e7",
		},
	}
	for i, tc := range testCases {
		h := lb.ChooseHost(tc.ctx)
		if h == nil {
			t.Errorf("#%d choose host failed", i)
			continue
		}
		if h.Hostname() != tc.expectedHost {
			t.Errorf("#%d choose host is not expected, expected %s, got %s", tc.expectedHost, h.Hostname())
		}
	}

}

// TestFallbackWithAllHosts configure all hosts as fallback, without default subset
func TestFallbackWithAllHosts(t *testing.T) {
	// fallback policy is any point
	// all host (no matter what lable it is) is used to fallback subset
	// for test simple, we use some simple host configs
	hosts := []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Hostname: "host1",
				Address:  "127.0.0.1:8080",
			},
			MetaData: map[string]string{
				"zone": "zone0",
				"room": "room0",
			},
		},
		{
			HostConfig: v2.HostConfig{
				Hostname: "host2",
				Address:  "127.0.0.1:8081",
			},
			MetaData: map[string]string{
				"zone": "zone1",
			},
		},
		{
			HostConfig: v2.HostConfig{
				Hostname: "host3",
				Address:  "127.0.0.1:8082",
			},
			MetaData: map[string]string{
				"room": "room1",
			},
		},
	}
	ps := createPrioritySet(hosts)
	// only create subset with version and xlarge
	// if not matched, use default subset: stage:dev
	cfg := &v2.LBSubsetConfig{
		FallBackPolicy: uint8(types.AnyEndPoint),
		SubsetSelectors: [][]string{
			[]string{
				"zone", "room",
			},
		},
	}
	//
	expectedResult := map[string][]string{
		"room->room0->zone->zone0->": []string{"host1"},
	}
	// New
	lb := NewSubsetLoadBalancer(types.RoundRobin, ps, newClusterStats("TestFallbackWithAllHosts"), NewLBSubsetInfo(cfg))
	if lb == nil {
		t.Fatal("create subset lb failed")
	}
	subSet := lb.(*subSetLoadBalancer).subSets
	result := &subSetMapResult{
		result: map[string][]string{},
	}
	// Verify subset created
	result.RangeSubsetMap("", subSet)
	if !resultMapEqual(result.result, expectedResult) {
		t.Errorf("subset tree created is not expected, got: %v, expected: %v", result.result, expectedResult)
	}
	// choose host test
	testCases := []struct {
		ctx           types.LoadBalancerContext
		expectedHosts []string
	}{
		{
			ctx: newMockLbContext(map[string]string{
				"zone": "zone0",
				"room": "room0",
			}),
			expectedHosts: []string{
				"host1", // matched subset
			},
		},
		// fallback
		{
			ctx: newMockLbContext(map[string]string{
				"zone": "zone0",
			}),
			expectedHosts: []string{
				"host1", "host2", "host3",
			},
		},
		{
			ctx: newMockLbContext(map[string]string{
				"room": "room0",
			}),
			expectedHosts: []string{
				"host1", "host2", "host3",
			},
		},
		{
			ctx: newMockLbContext(map[string]string{
				"zone": "zone0",
				"room": "room1",
			}),
			expectedHosts: []string{
				"host1", "host2", "host3",
			},
		},
		{
			ctx: newMockLbContext(map[string]string{
				"zone": "zone1",
			}),
			expectedHosts: []string{
				"host1", "host2", "host3",
			},
		},
		{
			ctx: newMockLbContext(map[string]string{
				"room": "room1",
			}),
			expectedHosts: []string{
				"host1", "host2", "host3",
			},
		},
		{
			ctx: newMockLbContext(map[string]string{}),
			expectedHosts: []string{
				"host1", "host2", "host3",
			},
		},
		{
			ctx: nil,
			expectedHosts: []string{
				"host1", "host2", "host3",
			},
		},
		{
			ctx: newMockLbContext(nil),
			expectedHosts: []string{
				"host1", "host2", "host3",
			},
		},
	}
RUNCASE:
	for i, tc := range testCases {
		for j := 0; j < 3; j++ { // choose multi times
			h := lb.ChooseHost(tc.ctx)
			if h == nil {
				t.Errorf("#%d choose host failed", i)
				continue RUNCASE
			}
			if !strInSlice(h.Hostname(), tc.expectedHosts) {
				t.Errorf("#%d choose host not expected, expected: %v, got: %s", tc.expectedHosts, h.Hostname())
				continue RUNCASE
			}
		}
	}
}

// TestDynamicSubsetHost with subset
// If a subset's hosts are all deleted, the subset map still exixts, but will returns a nil host
// If a new host with label is added, a new subset map will be created
func TestDynamicSubsetHost(t *testing.T) {
	// use cluster manager to register dynamic host changed
	clusterName := "TestSubset"
	hostA := v2.Host{
		HostConfig: v2.HostConfig{
			Address:  "127.0.0.1:8080",
			Hostname: "A",
		},
		MetaData: v2.Metadata{
			"zone":  "zone0",
			"group": "a",
		},
	}
	hosts := []v2.Host{
		hostA,
	}
	clusterConfig := v2.Cluster{
		Name:                 clusterName,
		ClusterType:          v2.SIMPLE_CLUSTER,
		LbType:               v2.LB_RANDOM,
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 1024,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy: uint8(types.AnyEndPoint),
			SubsetSelectors: [][]string{
				[]string{
					"zone", "group",
				},
				[]string{
					"zone",
				},
			},
		},
		Hosts: hosts,
	}
	clusters := []v2.Cluster{
		clusterConfig,
	}
	clusterMap := map[string][]v2.Host{
		clusterName: hosts,
	}
	// makes a cluster manager, and get the subset
	// reset clusrer manager
	clusterMangerInstance.Destory()
	cmi := NewClusterManager(nil, clusters, clusterMap, false, false)
	cm := cmi.(*clusterManager)
	v, _ := cm.primaryClusters.Load(clusterName)
	pc := v.(*primaryCluster)
	pcc := pc.cluster.(*simpleInMemCluster).cluster
	info := pcc.info
	lb := info.lbInstance.(*subSetLoadBalancer)
	// test
	// create a subset
	{
		expectedResult := map[string][]string{
			"group->a->zone->zone0->": []string{"A"},
			"zone->zone0->":           []string{"A"},
		}
		result := &subSetMapResult{
			result: map[string][]string{},
		}
		result.RangeSubsetMap("", lb.subSets)
		if !resultMapEqual(result.result, expectedResult) {
			t.Fatal("create subset is not expected", result.result)
		}
		// try to choose host, found A
		ctx := newMockLbContext(map[string]string{
			"zone":  "zone0",
			"group": "a",
		})
		// no fallback choose
		if h, ok := lb.TryChooseHostFromContext(ctx); !ok || h == nil || h.Hostname() != "A" {
			t.Fatal("choose host not expected")
		}

	}
	// remove a host the subset will be changed
	{
		cm.UpdateClusterHosts(clusterName, 0, []v2.Host{})
		// host removed, the tree still exists, but no more active
		expectedResult := map[string][]string{
			"group->a->zone->zone0->": []string{},
			"zone->zone0->":           []string{},
		}
		result := &subSetMapResult{
			result: map[string][]string{},
		}
		result.RangeSubsetMap("", lb.subSets)
		if !resultMapEqual(result.result, expectedResult) {
			t.Fatal("create subset is not expected", result.result)
		}
		// try to choose host, found nil
		ctx := newMockLbContext(map[string]string{
			"zone":  "zone0",
			"group": "a",
		})
		// with fallback choose, also get a nil
		if h := lb.ChooseHost(ctx); h != nil {
			t.Fatal("choost host not expected")
		}
	}
	// add a new host with new label, create a new subset
	{
		hostB := v2.Host{
			HostConfig: v2.HostConfig{
				Address:  "127.0.0.1:8080", // address is same as host A
				Hostname: "B",
			},
			MetaData: v2.Metadata{
				"zone":  "zone0",
				"group": "b",
			},
		}
		cm.UpdateClusterHosts(clusterName, 0, []v2.Host{hostB})
		expectedResult := map[string][]string{
			"group->a->zone->zone0->": []string{},
			"zone->zone0->":           []string{"B"},
			"group->b->zone->zone0->": []string{"B"},
		}
		result := &subSetMapResult{
			result: map[string][]string{},
		}
		result.RangeSubsetMap("", lb.subSets)
		if !resultMapEqual(result.result, expectedResult) {
			t.Fatal("create subset is not expected", result.result)
		}
		// try to choose host
		ctx := newMockLbContext(map[string]string{
			"zone":  "zone0",
			"group": "a",
		})
		// no fallback, found nothing
		if h, ok := lb.TryChooseHostFromContext(ctx); ok || h != nil {
			t.Fatal("choost host not expected")
		}
		// with fallback, found host
		if h := lb.ChooseHost(ctx); h == nil || h.Hostname() != "B" {
			t.Fatal("choost host not expected")
		}
		ctx2 := newMockLbContext(map[string]string{
			"zone":  "zone0",
			"group": "b",
		})
		// no fallback, found host
		if h, ok := lb.TryChooseHostFromContext(ctx2); !ok || h == nil || h.Hostname() != "B" {
			t.Fatal("choose host not expected")
		}
	}

}

func TestSubsetGetHostsNumber(t *testing.T) {
	ps := createPrioritySet(ExampleHostConfigs())
	lb := NewSubsetLoadBalancer(types.RoundRobin, ps, newClusterStats("TestSubsetGetHostsNumber"), NewLBSubsetInfo(ExampleSubsetConfig()))
	if lb == nil {
		t.Fatal("create subset lb failed")
	}
	sslb := lb.(*subSetLoadBalancer)
	testCases := []struct {
		ctx         types.LoadBalancerContext
		hostsNumber uint32
	}{
		{
			ctx: newMockLbContext(map[string]string{
				"stage": "dev",
				"type":  "std",
			}),
			hostsNumber: 1,
		},
		{
			ctx: newMockLbContext(map[string]string{
				"stage":   "dev",
				"version": "1.2-pre",
			}),
			hostsNumber: 1,
		},
		{
			ctx: newMockLbContext(map[string]string{
				"stage": "prod",
				"type":  "std",
			}),
			hostsNumber: 4,
		},
		{
			ctx: newMockLbContext(map[string]string{
				"stage": "prod",
				"type":  "bigmem",
			}),
			hostsNumber: 2,
		},
		{
			ctx: newMockLbContext(map[string]string{
				"stage":   "prod",
				"version": "1.0",
			}),
			hostsNumber: 3,
		},
		{
			ctx: newMockLbContext(map[string]string{
				"stage":   "prod",
				"version": "1.1",
			}),
			hostsNumber: 3,
		},
		{
			ctx: newMockLbContext(map[string]string{
				"version": "1.0",
			}),
			hostsNumber: 3,
		},

		{
			ctx: newMockLbContext(map[string]string{
				"version": "1.0",
				"xlarge":  "true",
			}),
			hostsNumber: 1,
		},

		{
			ctx: newMockLbContext(map[string]string{
				"version": "1.1",
			}),
			hostsNumber: 3,
		},

		{
			ctx: newMockLbContext(map[string]string{
				"version": "1.2-pre",
			}),
			hostsNumber: 1,
		},
	}
	for i, tc := range testCases {
		cnt := sslb.GetHostsNumber(tc.ctx.MetadataMatchCriteria())
		if cnt != tc.hostsNumber {
			t.Errorf("#%d get host number not expected, expected %d, got %d", i, tc.hostsNumber, cnt)
		}
	}
}

func TestGetFinalHost(t *testing.T) {
	pool := makePool(100)
	hosts := pool.MakeHosts(10)
	hsSubset := &hostSubsetImpl{
		hostSubset: &hostSet{
			hosts: hosts,
		},
	}
	hostsRemoved := make([]types.Host, 5)
	copy(hostsRemoved, hosts[5:])
	hostsAdded := pool.MakeHosts(5)
	final := types.SortedHosts(hsSubset.GetFinalHosts(hostsAdded, hostsRemoved))
	expected := types.SortedHosts(make([]types.Host, 10))
	copy(expected[:5], hosts[:5])
	copy(expected[5:], hostsAdded)
	sort.Sort(final)
	sort.Sort(expected)
	if !reflect.DeepEqual(final, expected) {
		t.Error("get final hosts unexpected")
	}
}

// utils for test
type mockLbContext struct {
	types.LoadBalancerContext
	mmc    types.MetadataMatchCriteria
	header types.HeaderMap
}

func newMockLbContext(m map[string]string) types.LoadBalancerContext {
	mmc := router.NewMetadataMatchCriteriaImpl(m)
	return &mockLbContext{
		mmc: mmc,
	}
}

func newMockLbContextWithHeader(m map[string]string, header types.HeaderMap) types.LoadBalancerContext {
	mmc := router.NewMetadataMatchCriteriaImpl(m)
	return &mockLbContext{
		mmc:    mmc,
		header: header,
	}
}

func (ctx *mockLbContext) MetadataMatchCriteria() types.MetadataMatchCriteria {
	return ctx.mmc
}

func (ctx *mockLbContext) DownstreamHeaders() types.HeaderMap {
	return ctx.header
}

func (ctx *mockLbContext) DownstreamContext() context.Context {
	return nil
}
