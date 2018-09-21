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

	"math"
	"math/rand"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func newHostV2(addr string, name string, weight uint32, meta v2.Metadata) v2.Host {
	return v2.Host{
		HostConfig: v2.HostConfig{
			Address:  addr,
			Hostname: name,
			Weight:   weight,
		},
		MetaData: meta,
	}
}

func Test_roundRobinLoadBalancer_ChooseHost(t *testing.T) {

	host1 := NewHost(newHostV2("127.0.0.1", "test", 0, nil), nil)
	host2 := NewHost(newHostV2("127.0.0.2", "test2", 0, nil), nil)
	host3 := NewHost(newHostV2("127.0.0.3", "test", 0, nil), nil)
	host4 := NewHost(newHostV2("127.0.0.4", "test2", 0, nil), nil)
	host5 := NewHost(newHostV2("127.0.0.5", "test2", 0, nil), nil)

	hosts1 := []types.Host{host1, host2}
	hosts2 := []types.Host{host3, host4}
	hosts3 := []types.Host{host5}

	hs1 := hostSet{
		hosts:        hosts1,
		healthyHosts: hosts1,
	}

	hs2 := hostSet{
		hosts:        hosts2,
		healthyHosts: hosts2,
	}

	hs3 := hostSet{
		hosts:        hosts3,
		healthyHosts: hosts3,
	}

	hostset := []types.HostSet{&hs1, &hs2, &hs3}

	prioritySet := prioritySet{
		hostSets: hostset,
	}

	loadbalaner := loadbalancer{
		prioritySet: &prioritySet,
	}

	l := &roundRobinLoadBalancer{
		loadbalancer: loadbalaner,
	}

	want := []types.Host{host1, host2, host3, host4, host5}

	for i := 0; i < len(want); i++ {
		got := l.ChooseHost(nil)
		if got != want[i] {
			t.Errorf("Test Error in case %d , got %+v, but want %+v,", i, got, want[i])
		}
	}
}

func TestSmoothWeightedRRLoadBalancer_ChooseHost(t *testing.T) {
	type testCase struct {
		lb types.LoadBalancer
	}

	host1 := NewHost(newHostV2("127.0.0.1", "a", 5, nil), nil)
	host2 := NewHost(newHostV2("127.0.0.2", "b", 3, nil), nil)
	host3 := NewHost(newHostV2("127.0.0.3", "c", 2, nil), nil)

	hosts1 := []types.Host{host1, host2, host3}
	hosts2 := []types.Host{host1, host2}
	hosts3 := []types.Host{host1}
	hosts4 := []types.Host{}

	hs1 := hostSet{
		hosts:        hosts1,
		healthyHosts: hosts1,
	}
	hs2 := hostSet{
		hosts:        hosts2,
		healthyHosts: hosts2,
	}
	hs3 := hostSet{
		hosts:        hosts3,
		healthyHosts: hosts3,
	}
	hs4 := hostSet{
		hosts:        hosts4,
		healthyHosts: hosts4,
	}
	hostset1 := []types.HostSet{&hs1}
	hostset2 := []types.HostSet{&hs2}
	hostset3 := []types.HostSet{&hs3}
	hostset4 := []types.HostSet{&hs4}

	tests := []struct {
		name string
		args *prioritySet
		want []float64
	}{
		{
			name: "fullTest",
			args: &prioritySet{
				hostSets: hostset1,
			},
			want: []float64{0.5, 0.3, 0.2},
		},
		{
			name: "case2",
			args: &prioritySet{
				hostSets: hostset2,
			},
			want: []float64{5.0 / 8.0, 3.0 / 8.0, 0},
		},
		{
			name: "case3",
			args: &prioritySet{
				hostSets: hostset3,
			},
			want: []float64{1.0, 0, 0},
		},
		{
			name: "zeroTest",
			args: &prioritySet{
				hostSets: hostset4,
			},
			want: []float64{0, 0, 0},
		},
	}

	thres := 0.01

	for _, tt := range tests {
		var a, b, c float64
		var i float64

		l1 := newSmoothWeightedRRLoadBalancer(tt.args)
		runningTimes := float64(rand.Int31n(1000))

		for ; i < runningTimes; i++ {
			host := l1.ChooseHost(nil)
			//	t.Log(host.Hostname())
			if host == nil {
				if tt.name == "zeroTest" {
					return
				}
				t.Errorf("test sommoth loalbalancer err, want a = %f, b = %f, c = %f,  got a = %f, b=%f, c=%f, case = %s", a/runningTimes, b/runningTimes, c/runningTimes,
					tt.want[0], tt.want[1], tt.want[2], tt.name)
			}

			switch host.Hostname() {
			case "a":
				a++
			case "b":
				b++
			case "c":
				c++
			}
		}

		if (a+b+c) != runningTimes || math.Abs(a/runningTimes-tt.want[0]) > thres || math.Abs(b/runningTimes-tt.want[1]) > thres || math.Abs(c/runningTimes-tt.want[2]) > thres {
			t.Errorf("test sommoth loalbalancer err, want a = %f, b = %f, c = %f,  got a = %f, b=%f, c=%f, case = %s", a/runningTimes, b/runningTimes, c/runningTimes,
				tt.want[0], tt.want[1], tt.want[2], tt.name)
		}
	}
}

func TestSmoothWeightedRRLoadBalancer_UpdateHost(t *testing.T) {

	host1 := NewHost(newHostV2("127.0.0.1", "a", 8, nil), nil)
	host2 := NewHost(newHostV2("127.0.0.2", "b", 2, nil), nil)
	host3 := NewHost(newHostV2("127.0.0.3", "c", 5, nil), nil)
	host4 := NewHost(newHostV2("127.0.0.4", "d", 5, nil), nil)

	hosts1 := []types.Host{host1, host2, host3}

	hs1 := hostSet{
		hosts:        hosts1,
		healthyHosts: hosts1,
	}

	hostset := []types.HostSet{&hs1}
	ps := &prioritySet{
		hostSets: hostset,
	}

	loadbBalancer := newSmoothWeightedRRLoadBalancer(ps)

	type args struct {
		healthyHosts []types.Host
		addedHosts   []types.Host
		removedHosts []types.Host
	}

	tests := []struct {
		name string
		args args
		want []float64
	}{
		{
			name: "removeTest",
			args: args{
				healthyHosts: []types.Host{host1, host2},
				addedHosts:   nil,
				removedHosts: []types.Host{host3},
			},
			want: []float64{8.0 / 10.0, 2.0 / 10.0, 0.0, 0.0},
		},
		{
			name: "addTest",
			args: args{
				healthyHosts: []types.Host{host1, host2, host3, host4},
				addedHosts:   []types.Host{host3, host4},
				removedHosts: nil,
			},
			want: []float64{8.0 / 20.0, 2.0 / 20.0, 5.0 / 20.0, 5.0 / 20.0},
		},
		{
			name: "zeroTest",
			args: args{
				healthyHosts: []types.Host{},
				addedHosts:   nil,
				removedHosts: []types.Host{host1, host2, host3, host4},
			},
			want: []float64{0.0, 0.0, 0.0, 0.0},
		},
		{
			name: "fullTest",
			args: args{
				healthyHosts: []types.Host{host1, host2, host3, host4},
				addedHosts:   []types.Host{host1, host2, host3, host4},
				removedHosts: nil,
			},
			want: []float64{8.0 / 20.0, 2.0 / 20.0, 5.0 / 20.0, 5.0 / 20.0},
		},
	}

	for _, tt := range tests {
		// update healthy-hosts
		ps.hostSets = []types.HostSet{&hostSet{healthyHosts: tt.args.healthyHosts}}

		if ll, ok := loadbBalancer.(*smoothWeightedRRLoadBalancer); ok {
			ll.UpdateHost(0, tt.args.addedHosts, tt.args.removedHosts)
		}

		runningTimes := float64(rand.Int31n(1000))
		var a, b, c, d float64
		var i float64

		for ; i < runningTimes; i++ {
			host := loadbBalancer.ChooseHost(nil)

			if host == nil {
				if tt.name == "zeroTest" {
					return
				}
				t.Errorf("test sommoth loalbalancer err, want a = %f, b = %f, c = %f,  got a = %f, b=%f, c=%f, case = %s", a/runningTimes, b/runningTimes, c/runningTimes,
					tt.want[0], tt.want[1], tt.want[2], tt.name)
			}

			switch host.Hostname() {
			case "a":
				a++
			case "b":
				b++
			case "c":
				c++
			case "d":
				d++
			}
		}

		thres := 0.1

		if (a+b+c+d) != runningTimes || math.Abs(a/runningTimes-tt.want[0]) > thres || math.Abs(b/runningTimes-tt.want[1]) > thres ||
			math.Abs(c/runningTimes-tt.want[2]) > thres || math.Abs(d/runningTimes-tt.want[3]) > thres {
			t.Errorf("test sommoth loalbalancer err, want a = %f, b = %f, c = %f,d=%f,  got a = %f, b=%f, c=%f, d = %f, case = %s", a/runningTimes, b/runningTimes, c/runningTimes,
				d/runningTimes, tt.want[0], tt.want[1], tt.want[2], tt.want[3], tt.name)
		}
	}
}

func MockRouter(names []string) v2.Router {
	r := v2.Router{}
	if len(names) < 2 {
		return r
	}
	r.Match = v2.RouterMatch{
		Headers: []v2.HeaderMatcher{
			v2.HeaderMatcher{Name: "service", Value: ".*"},
		},
	}
	r.Route = v2.RouteAction{
		RouterActionConfig: v2.RouterActionConfig{
			ClusterName: names[0],
			WeightedClusters: []v2.WeightedCluster{
				{
					Cluster: v2.ClusterWeight{
						ClusterWeightConfig: v2.ClusterWeightConfig{
							Name:   names[0],
							Weight: 60,
						},
						MetadataMatch: map[string]string{"label": "blue"},
					},
				},
				{
					Cluster: v2.ClusterWeight{
						ClusterWeightConfig: v2.ClusterWeightConfig{
							Name:   names[1],
							Weight: 40,
						},
						MetadataMatch: map[string]string{"label": "green"},
					},
				},
			},
		},
	}
	return r
}

func MockRouterMatcher() (types.Routers, error) {
	virtualHosts := []*v2.VirtualHost{
		&v2.VirtualHost{Domains: []string{"www.alibaba.com"}, Routers: []v2.Router{MockRouter([]string{"c1", "c2"})}},
		&v2.VirtualHost{Domains: []string{"www.antfin.com"}, Routers: []v2.Router{MockRouter([]string{"a1", "a2"})}},
	}
	cfg := &v2.Proxy{
		VirtualHosts: virtualHosts,
	}

	return router.NewRouteMatcher(cfg)
}

func mockClusterManager() types.ClusterManager {
	host1 := newHostV2("127.0.0.1", "h1", 5, v2.Metadata{"label": "blue"})
	host2 := newHostV2("127.0.0.2", "h2", 5, v2.Metadata{"label": "blue"})
	host3 := newHostV2("127.0.0.3", "h3", 5, v2.Metadata{"label": "green"})
	host4 := newHostV2("127.0.0.4", "h4", 5, v2.Metadata{"label": "green"})
	host5 := newHostV2("127.0.0.5", "h5", 5, v2.Metadata{"label": "blue"})
	host6 := newHostV2("127.0.0.6", "h6", 5, v2.Metadata{"label": "blue"})
	host7 := newHostV2("127.0.0.7", "h7", 5, v2.Metadata{"label": "green"})
	host8 := newHostV2("127.0.0.8", "h8", 5, v2.Metadata{"label": "green"})

	clusters := []v2.Cluster{
		{
			Name:        "c1",
			ClusterType: v2.SIMPLE_CLUSTER,
			LBSubSetConfig: v2.LBSubsetConfig{
				FallBackPolicy:  1,
				DefaultSubset:   map[string]string{"label": "blue"},
				SubsetSelectors: [][]string{{"label"}},
			},
			LbType: v2.LB_ROUNDROBIN,
			Hosts:  []v2.Host{host1, host2},
		},
		{
			Name:        "c2",
			ClusterType: v2.SIMPLE_CLUSTER,
			LBSubSetConfig: v2.LBSubsetConfig{
				FallBackPolicy:  1,
				DefaultSubset:   map[string]string{"label": "green"},
				SubsetSelectors: [][]string{{"label"}},
			},
			LbType: v2.LB_ROUNDROBIN,
			Hosts:  []v2.Host{host3, host4},
		},
		{
			Name:        "a1",
			ClusterType: v2.SIMPLE_CLUSTER,
			LBSubSetConfig: v2.LBSubsetConfig{
				FallBackPolicy:  1,
				DefaultSubset:   map[string]string{"label": "blue"},
				SubsetSelectors: [][]string{{"label"}},
			},
			LbType: v2.LB_ROUNDROBIN,
			Hosts:  []v2.Host{host5, host6},
		},
		{
			Name:        "a2",
			ClusterType: v2.SIMPLE_CLUSTER,
			LBSubSetConfig: v2.LBSubsetConfig{
				FallBackPolicy:  1,
				DefaultSubset:   map[string]string{"label": "green"},
				SubsetSelectors: [][]string{{"label"}},
			},
			LbType: v2.LB_ROUNDROBIN,
			Hosts:  []v2.Host{host7, host8},
		},
	}

	clusterMap := map[string][]v2.Host{
		"c1": []v2.Host{host1, host2},
		"c2": []v2.Host{host3, host4},
		"a1": []v2.Host{host5, host6},
		"a2": []v2.Host{host7, host8},
	}

	return NewClusterManager(nil, clusters, clusterMap, true, false)
}

func Benchmark_RouteAndLB(b *testing.B) {

	mockedHeader := map[string]string{
		strings.ToLower(protocol.MosnHeaderHostKey): "www.alibaba.com",
		"service": "test",
	}

	mockedClusterMng := mockClusterManager().(*clusterManager)
	mockedRouter, err := MockRouterMatcher()
	if err != nil {
		b.Errorf(err.Error())
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		route := mockedRouter.Route(mockedHeader, 1)
		if route == nil {
			b.Errorf("%s match failed\n", "www.alibaba.com")
			return
		}

		clustername := route.RouteRule().ClusterName()
		clusterSnapshot := mockedClusterMng.getOrCreateClusterSnapshot(clustername)

		if clusterSnapshot == nil {
			b.Errorf("Cluster is nil, cluster name = %s", clustername)
			return
		}

		if mmc, ok := route.RouteRule().MetadataMatchCriteria(clustername).(*router.MetadataMatchCriteriaImpl); ok {
			ctx := &ContextImplMock{
				mmc: mmc,
			}

			host := clusterSnapshot.LoadBalancer().ChooseHost(ctx)
			b.Logf("host name = %s", host.Hostname())
		} else {
			b.Errorf(" host select error, clusterName=%s", clustername)
		}
	}
}
