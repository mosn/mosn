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
	"net"
	"reflect"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// we test following cases form envoy's example
/*
`stage=prod, type=std` (e1, e2, e3, e4)      ------> case1
`stage=prod, type=bigmem` (e5, e6)
`stage=dev, type=std` (e7)
`stage=prod, version=1.0` (e1, e2, e5)
`stage=prod, version=1.1` (e3, e4, e6)
`stage=dev, version=1.2-pre` (e7)
`version=1.0` (e1, e2, e5)
`version=1.1` (e3, e4, e6)
`version=1.2-pre` (e7)
`version=1.0, xlarge=true` (e1)  ------> case10
*/

/*
`stage=prod, type=std, version=1.0` (e1, e2)    -------> fallback subset
*/

func init() {
	log.InitDefaultLogger("", log.DEBUG)
}

var prioritySetExample = prioritySet{
	hostSets: []types.HostSet{
		&hostSet{
			hosts: InitExampleHosts(),
		},
	},
}

var DefaultSubset = map[string]string{
	"stage":   "prod",
	"version": "1.0",
	"type":    "std",
}

var SubsetSelectors = [][]string{
	{"stage", "type"},
	{"stage", "version"},
	{"version"},
	{"xlarge", "version"},
}

var SubsetLbExample = subSetLoadBalancer{
	fallBackPolicy:        2,
	stats:                 newClusterStats(v2.Cluster{Name: "testcluster"}),
	lbType:                types.RoundRobin,
	originalPrioritySet:   &prioritySetExample,
	defaultSubSetMetadata: InitDefaultSubsetMetadata(),
	subSetKeys:            GenerateSubsetKeys(SubsetSelectors),
}

// passed
// test fallback subset creation
func Test_subSetLoadBalancer_UpdateFallbackSubset(t *testing.T) {

	hostSet := InitExampleHosts()
	type args struct {
		priority     uint32
		hostAdded    []types.Host
		hostsRemoved []types.Host
	}
	tests := []struct {
		name string
		args args
		want []types.Host
	}{
		{
			name: "case1",
			args: args{
				priority:     0,
				hostAdded:    SubsetLbExample.originalPrioritySet.HostSetsByPriority()[0].Hosts(),
				hostsRemoved: nil,
			},
			want: []types.Host{
				hostSet[0], hostSet[1],
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			sslb := SubsetLbExample
			sslb.UpdateFallbackSubset(tt.args.priority, tt.args.hostAdded, tt.args.hostsRemoved)

			for idx, host := range sslb.fallbackSubset.prioritySubset.GetOrCreateHostSubset(0).Hosts() {

				if host.Hostname() != tt.want[idx].Hostname() {
					t.Errorf("Test_subSetLoadBalancer_UpdateFallbackSubset Error, got = %v, want %v", host,
						tt.want[idx])
				}
			}

		})
	}
}

// passed
// test subset map creation and subset search
func Test_subSetLoadBalancer_ProcessSubsets(t *testing.T) {

	hostSet := InitExampleHosts()
	updateCb := func(entry types.LBSubsetEntry) {
		entry.PrioritySubset().Update(0, SubsetLbExample.originalPrioritySet.HostSetsByPriority()[0].Hosts(), nil)
	}

	newCb := func(entry types.LBSubsetEntry, predicate types.HostPredicate, kvs types.SubsetMetadata, addinghost bool) {
		if addinghost {
			prioritySubset := NewPrioritySubsetImpl(&SubsetLbExample, predicate)
			entry.SetPrioritySubset(prioritySubset)
		}
	}

	hostadded := SubsetLbExample.originalPrioritySet.HostSetsByPriority()[0].Hosts()

	type args struct {
		hostAdded     []types.Host
		hostsRemoved  []types.Host
		updateCB      func(types.LBSubsetEntry)
		newCB         func(types.LBSubsetEntry, types.HostPredicate, types.SubsetMetadata, bool)
		matchCriteria []types.MetadataMatchCriterion
	}

	tests := []struct {
		name string
		args args
		want []types.Host
	}{
		{
			name: "case1",
			args: args{
				hostAdded:    hostadded,
				hostsRemoved: nil,
				updateCB:     updateCb,
				newCB:        newCb,
				matchCriteria: []types.MetadataMatchCriterion{
					&router.MetadataMatchCriterionImpl{
						"stage",
						types.GenerateHashedValue("prod"),
					},
					&router.MetadataMatchCriterionImpl{
						"type",
						types.GenerateHashedValue("std"),
					},
				},
			},
			want: []types.Host{
				hostSet[0], //e1,e2,e3,e4
				hostSet[1], hostSet[2], hostSet[3],
			},
		},

		{
			name: "case2",
			args: args{
				hostAdded:    hostadded,
				hostsRemoved: nil,
				updateCB:     updateCb,
				newCB:        newCb,
				matchCriteria: []types.MetadataMatchCriterion{
					&router.MetadataMatchCriterionImpl{
						"stage",
						types.GenerateHashedValue("prod"),
					},
					&router.MetadataMatchCriterionImpl{
						"type",
						types.GenerateHashedValue("bigmem"),
					},
				},
			},
			want: []types.Host{
				hostSet[4], //e5,e6
				hostSet[5],
			},
		},
		{
			name: "case3",
			args: args{
				hostAdded:    hostadded,
				hostsRemoved: nil,
				updateCB:     updateCb,
				newCB:        newCb,
				matchCriteria: []types.MetadataMatchCriterion{
					&router.MetadataMatchCriterionImpl{
						"stage",
						types.GenerateHashedValue("dev"),
					},
					&router.MetadataMatchCriterionImpl{
						"type",
						types.GenerateHashedValue("std"),
					},
				},
			},
			want: []types.Host{
				hostSet[6], //e5,e6
			},
		},
		{
			name: "case4",
			args: args{
				hostAdded:    hostadded,
				hostsRemoved: nil,
				updateCB:     updateCb,
				newCB:        newCb,
				matchCriteria: []types.MetadataMatchCriterion{
					&router.MetadataMatchCriterionImpl{
						"stage",
						types.GenerateHashedValue("prod"),
					},
					&router.MetadataMatchCriterionImpl{
						"version",
						types.GenerateHashedValue("1.0"),
					},
				},
			},
			want: []types.Host{
				hostSet[0], //e1 e2 e5
				hostSet[1],
				hostSet[4],
			},
		},

		{
			name: "case5",
			args: args{
				hostAdded:    hostadded,
				hostsRemoved: nil,
				updateCB:     updateCb,
				newCB:        newCb,
				matchCriteria: []types.MetadataMatchCriterion{
					&router.MetadataMatchCriterionImpl{
						"stage",
						types.GenerateHashedValue("prod"),
					},
					&router.MetadataMatchCriterionImpl{
						"version",
						types.GenerateHashedValue("1.1"),
					},
				},
			},
			want: []types.Host{
				hostSet[2], //e3 e4 e6
				hostSet[3],
				hostSet[5],
			},
		},

		{
			name: "case6",
			args: args{
				hostAdded:    hostadded,
				hostsRemoved: nil,
				updateCB:     updateCb,
				newCB:        newCb,
				matchCriteria: []types.MetadataMatchCriterion{
					&router.MetadataMatchCriterionImpl{
						"stage",
						types.GenerateHashedValue("dev"),
					},
					&router.MetadataMatchCriterionImpl{
						"version",
						types.GenerateHashedValue("1.2-pre"),
					},
				},
			},
			want: []types.Host{
				hostSet[6], //e7

			},
		},

		{
			name: "case7",
			args: args{
				hostAdded:    hostadded,
				hostsRemoved: nil,
				updateCB:     updateCb,
				newCB:        newCb,
				matchCriteria: []types.MetadataMatchCriterion{
					&router.MetadataMatchCriterionImpl{
						"version",
						types.GenerateHashedValue("1.0"),
					},
				},
			},
			want: []types.Host{
				hostSet[0], //e1,e2,e5
				hostSet[1],
				hostSet[4],
			},
		},

		{
			name: "case8",
			args: args{
				hostAdded:    hostadded,
				hostsRemoved: nil,
				updateCB:     updateCb,
				newCB:        newCb,
				matchCriteria: []types.MetadataMatchCriterion{
					&router.MetadataMatchCriterionImpl{
						"version",
						types.GenerateHashedValue("1.1"),
					},
				},
			},
			want: []types.Host{
				hostSet[2], //e3 e4 e6
				hostSet[3],
				hostSet[5],
			},
		},

		{
			name: "case9",
			args: args{
				hostAdded:    hostadded,
				hostsRemoved: nil,
				updateCB:     updateCb,
				newCB:        newCb,
				matchCriteria: []types.MetadataMatchCriterion{
					&router.MetadataMatchCriterionImpl{
						"version",
						types.GenerateHashedValue("1.2-pre"),
					},
				},
			},
			want: []types.Host{
				hostSet[6], //e7
			},
		},

		{
			name: "testcase10",
			args: args{
				hostAdded:    SubsetLbExample.originalPrioritySet.HostSetsByPriority()[0].Hosts(),
				hostsRemoved: nil,
				updateCB:     updateCb,
				newCB:        newCb,
				matchCriteria: []types.MetadataMatchCriterion{
					&router.MetadataMatchCriterionImpl{
						"version",
						types.GenerateHashedValue("1.0"),
					},
					&router.MetadataMatchCriterionImpl{
						"xlarge",
						types.GenerateHashedValue("true"),
					},
				},
			},
			want: []types.Host{
				hostSet[0], // e1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sslb := SubsetLbExample

			sslb.subSets = make(map[string]types.ValueSubsetMap)

			sslb.ProcessSubsets(tt.args.hostAdded, tt.args.hostsRemoved, tt.args.updateCB, tt.args.newCB)

			for idx, host := range sslb.FindSubset(tt.args.matchCriteria).PrioritySubset().GetOrCreateHostSubset(0).Hosts() {
				if host.Hostname() != tt.want[idx].Hostname() {
					t.Errorf("subSetLoadBalancer.ChooseHost() = %v, want %v", host.Hostname(), tt.want[idx].Hostname())
				}
			}
		})
	}
}

func Test_subSetLoadBalancer_ChooseHost(t *testing.T) {
	hostSet := InitExampleHosts()

	type args struct {
		context types.LoadBalancerContext
	}

	tests := []struct {
		name string
		args args
		want types.Host
	}{
		{
			name: "case1",
			args: args{
				context: &ContextImplMock{
					mmc: &router.MetadataMatchCriteriaImpl{
						MatchCriteriaArray: []types.MetadataMatchCriterion{
							&router.MetadataMatchCriterionImpl{
								"stage",
								types.GenerateHashedValue("prod"),
							},
							&router.MetadataMatchCriterionImpl{
								"type",
								types.GenerateHashedValue("std"),
							},
						},
					},
				},
			},
			want: hostSet[0],
		},

		{
			name: "testdefault",
			args: args{
				context: &ContextImplMock{
					mmc: &router.MetadataMatchCriteriaImpl{
						MatchCriteriaArray: []types.MetadataMatchCriterion{
							&router.MetadataMatchCriterionImpl{
								"stage",
								types.GenerateHashedValue("prod"),
							},
							&router.MetadataMatchCriterionImpl{
								"type",
								types.GenerateHashedValue("unknown"),
							},
						},
					},
				},
			},
			want: hostSet[0], // default host: e1, e2
		},
	}

	sslb := NewSubsetLoadBalancer(types.RoundRobin, &prioritySetExample,
		newClusterStats(v2.Cluster{Name: "testcluster"}), NewLBSubsetInfo(InitExampleLbSubsetConfig()))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sslb.ChooseHost(tt.args.context)
			if got.AddressString() != tt.want.AddressString() {
				t.Errorf("subSetLoadBalancer.ChooseHost() = %v, want %v", got.AddressString(), tt.want.AddressString())
			}
		})
	}
}

// passed
func TestGenerateSubsetKeys(t *testing.T) {
	type args struct {
		keysArray [][]string
	}

	tests := []struct {
		name string
		args args
		want [][]string
	}{
		{
			name: "test",
			args: args{
				keysArray: [][]string{
					{"stage", "type"},
					{"stage", "version"},
					{"version"},
					{"xlarge", "version"},
				},
			},
			want: [][]string{
				{"stage", "type"},
				{"stage", "version"},
				{"version"},
				{"version", "xlarge"},
			},
		},

		{
			name: "test2",
			args: args{
				keysArray: [][]string{
					{"stage", "type", "type"},
					{"stage", "version"},
					{"stage", "version"},
					{"xlarge", "version"},
				},
			},
			want: [][]string{
				{"stage", "type"},
				{"stage", "version"},
				{"version", "xlarge"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateSubsetKeys(tt.args.keysArray); got != nil {
				for index, g := range got {
					for idx, key := range g.Keys() {
						if tt.want[index][idx] != key {
							t.Errorf("TestGenerateSubsetKeys Error, got = %v, want %v", got, tt.want)
						}
					}
				}
			}

		})
	}
}

// passed
func TestGenerateDftSubsetKeys(t *testing.T) {
	type args struct {
		dftkeys types.SortedMap
	}

	tests := []struct {
		name string
		args args
		want types.SubsetMetadata
	}{
		{
			name: "test1",
			args: args{
				dftkeys: []types.SortedPair{
					{"stage", "prod"},
					{"type", "std"},
					{"version", "1.0"},
				},
			},
			want: InitDefaultSubsetMetadata(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateDftSubsetKeys(tt.args.dftkeys); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateDftSubsetKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func InitDefaultSubsetMetadata() types.SubsetMetadata {

	p1 := types.Pair{"stage", types.GenerateHashedValue("prod")}

	p2 := types.Pair{"type", types.GenerateHashedValue("std")}

	p3 := types.Pair{"version", types.GenerateHashedValue("1.0")}

	return []types.Pair{p1, p2, p3}
}

func InitExampleLbSubsetConfig() *v2.LBSubsetConfig {
	var lbsubsetconfig = &v2.LBSubsetConfig{

		FallBackPolicy: 2, //"DEFAULT_SUBSET"
		DefaultSubset:  DefaultSubset,

		SubsetSelectors: SubsetSelectors,
	}

	return lbsubsetconfig
}

func InitExampleHosts() []types.Host {
	var HostAddress = []string{"127.0.0.1:0001", "127.0.0.1:0002", "127.0.0.1:0003",
		"127.0.0.1:0004", "127.0.0.1:0005", "127.0.0.1:0006", "127.0.0.1:0007"}
	var hosts []types.Host

	e1 := v2.Host{
		HostConfig: v2.HostConfig{
			Hostname: "e1",
			Address:  HostAddress[0],
			Weight:   100,
		},
		MetaData: map[string]string{
			"stage":   "prod",
			"version": "1.0",
			"type":    "std",
			"xlarge":  "true", // Note: currently, we only support value = string
		},
	}
	hosts = append(hosts, &host{hostInfo: newHostInfo(nil, e1, nil)})

	e2 := v2.Host{
		HostConfig: v2.HostConfig{
			Hostname: "e2",
			Address:  HostAddress[1],
			Weight:   100,
		},
		MetaData: map[string]string{
			"stage":   "prod",
			"version": "1.0",
			"type":    "std",
		},
	}
	hosts = append(hosts, &host{hostInfo: newHostInfo(nil, e2, nil)})

	e3 := v2.Host{
		HostConfig: v2.HostConfig{
			Hostname: "e3",
			Address:  HostAddress[2],
			Weight:   100,
		},
		MetaData: map[string]string{
			"stage":   "prod",
			"version": "1.1",
			"type":    "std",
		},
	}
	hosts = append(hosts, &host{hostInfo: newHostInfo(nil, e3, nil)})

	e4 := v2.Host{
		HostConfig: v2.HostConfig{
			Hostname: "e4",
			Address:  HostAddress[3],
			Weight:   100,
		},
		MetaData: map[string]string{
			"stage":   "prod",
			"version": "1.1",
			"type":    "std",
		},
	}
	hosts = append(hosts, &host{hostInfo: newHostInfo(nil, e4, nil)})

	e5 := v2.Host{
		HostConfig: v2.HostConfig{
			Hostname: "e5",
			Address:  HostAddress[4],
			Weight:   100,
		},
		MetaData: map[string]string{
			"stage":   "prod",
			"version": "1.0",
			"type":    "bigmem",
		},
	}
	hosts = append(hosts, &host{hostInfo: newHostInfo(nil, e5, nil)})

	e6 := v2.Host{
		HostConfig: v2.HostConfig{
			Hostname: "e6",
			Address:  HostAddress[5],
			Weight:   100,
		},
		MetaData: map[string]string{
			"stage":   "prod",
			"version": "1.1",
			"type":    "bigmem",
		},
	}
	hosts = append(hosts, &host{hostInfo: newHostInfo(nil, e6, nil)})

	e7 := v2.Host{
		HostConfig: v2.HostConfig{
			Hostname: "e7",
			Address:  HostAddress[6],
			Weight:   100,
		},
		MetaData: map[string]string{
			"stage":   "dev",
			"version": "1.2-pre",
			"type":    "std",
		},
	}
	hosts = append(hosts, &host{hostInfo: newHostInfo(nil, e7, nil)})

	return hosts
}

func TestWeightedClusterRoute(t *testing.T) {
	routerMock1 := &v2.Router{}

	routerMock1.Route = v2.RouteAction{
		RouterActionConfig: v2.RouterActionConfig{
			ClusterName: "defaultCluster",
			WeightedClusters: []v2.WeightedCluster{
				{
					Cluster: v2.ClusterWeight{
						ClusterWeightConfig: v2.ClusterWeightConfig{
							Name:   "w1",
							Weight: 90,
						},
						MetadataMatch: map[string]string{
							"version": "v1"},
					},
				},

				{
					Cluster: v2.ClusterWeight{
						ClusterWeightConfig: v2.ClusterWeightConfig{
							Name:   "w2",
							Weight: 10,
						},
						MetadataMatch: map[string]string{
							"version": "v2"},
					},
				},
			},
		},
	}

	routeRuleImplBase, _ := router.NewRouteRuleImplBase(nil, routerMock1)
	clustername := routeRuleImplBase.ClusterName()

	if clustername == "w1" {
		if weightedClusterEntry, ok := routeRuleImplBase.WeightedCluster()[clustername]; ok {
			metadataMatchCriteria := weightedClusterEntry.GetClusterMetadataMatchCriteria()
			sslb := NewSubsetLoadBalancer(types.RoundRobin, &priorityMock,
				newClusterStats(v2.Cluster{Name: "w1"}), NewLBSubsetInfo(SubsetMock()))

			context := &ContextImplMock{
				mmc: metadataMatchCriteria,
			}

			if host := sslb.ChooseHost(context); host.Hostname() != "e1" {
				t.Errorf("routing with weighted cluster error, want e1, but got:%s", host.Hostname())
			}
		} else {
			t.Errorf("routing with weighted cluster error, no clustername found")
		}
	} else if clustername == "w2" {
		if weightedClusterEntry, ok := routeRuleImplBase.WeightedCluster()[clustername]; ok {
			metadataMatchCriteria := weightedClusterEntry.GetClusterMetadataMatchCriteria()
			sslb := NewSubsetLoadBalancer(types.RoundRobin, &priorityMock,
				newClusterStats(v2.Cluster{Name: "w2"}), NewLBSubsetInfo(SubsetMock()))

			context := &ContextImplMock{
				mmc: metadataMatchCriteria,
			}

			if host := sslb.ChooseHost(context); host.Hostname() != "e2" {
				t.Errorf("routing with weighted cluster error, want e1, but got:%s", host.Hostname())
			}
		} else {
			t.Errorf("routing with weighted cluster error, no clustername found")
		}
	} else {
		t.Errorf("routing with weighted cluster error, no clustername found")
	}
}

var priorityMock = prioritySet{
	hostSets: []types.HostSet{
		&hostSet{
			hosts: HostsMock(),
		},
	},
}

func SubsetMock() *v2.LBSubsetConfig {
	lbsubsetconfig := &v2.LBSubsetConfig{
		FallBackPolicy:  2, //"DEFAULT_SUBSET"
		SubsetSelectors: [][]string{{"version"}},
	}

	return lbsubsetconfig
}

func HostsMock() []types.Host {
	var hosts []types.Host

	e1 := v2.Host{
		HostConfig: v2.HostConfig{
			Hostname: "e1",
			Weight:   50,
		},
		MetaData: map[string]string{
			"version": "v1",
		},
	}
	hosts = append(hosts, &host{hostInfo: newHostInfo(nil, e1, nil)})

	e2 := v2.Host{
		HostConfig: v2.HostConfig{
			Hostname: "e2",
			Weight:   50,
		},
		MetaData: map[string]string{
			"version": "v2",
		},
	}
	//
	hosts = append(hosts, &host{hostInfo: newHostInfo(nil, e2, nil)})

	return hosts
}

type ContextImplMock struct {
	mmc *router.MetadataMatchCriteriaImpl
}

func (ci *ContextImplMock) ComputeHashKey() types.HashedValue {
	//return [16]byte{}
	return ""
}

func (ci *ContextImplMock) MetadataMatchCriteria() types.MetadataMatchCriteria {
	return ci.mmc
}

func (ci *ContextImplMock) DownstreamConnection() net.Conn {
	return nil
}

func (ci *ContextImplMock) DownstreamHeaders() map[string]string {
	return nil
}
