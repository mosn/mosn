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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var host1 = v2.Host{
	HostConfig: v2.HostConfig{
		Address: "127.0.0.1", Hostname: "h1", Weight: 5,
	},
	MetaData: v2.Metadata{"label": "blue"},
}
var host2 = v2.Host{
	HostConfig: v2.HostConfig{
		Address: "127.0.0.2", Hostname: "h2", Weight: 5,
	},
	MetaData: v2.Metadata{"label": "blue"},
}
var host3 = v2.Host{
	HostConfig: v2.HostConfig{
		Address: "127.0.0.3", Hostname: "h3", Weight: 5,
	},
	MetaData: v2.Metadata{"label": "green"},
}
var host4 = v2.Host{
	HostConfig: v2.HostConfig{
		Address: "127.0.0.4", Hostname: "h4", Weight: 5,
	},
	MetaData: v2.Metadata{"label": "green"},
}

var clusterOrigin = v2.Cluster{
	Name:        "o1",
	ClusterType: v2.SIMPLE_CLUSTER,
	LBSubSetConfig: v2.LBSubsetConfig{
		FallBackPolicy:  1,
		DefaultSubset:   map[string]string{"label": "blue"},
		SubsetSelectors: [][]string{{"label"}},
	},
	LbType: v2.LB_ROUNDROBIN,
	Hosts:  []v2.Host{host1, host2},
}

var clusterOrigin2 = v2.Cluster{
	Name:        "o2",
	ClusterType: v2.SIMPLE_CLUSTER,
	LBSubSetConfig: v2.LBSubsetConfig{
		FallBackPolicy:  1,
		DefaultSubset:   map[string]string{"label": "green"},
		SubsetSelectors: [][]string{{"label"}},
	},
	LbType: v2.LB_ROUNDROBIN,
	Hosts:  []v2.Host{host3, host4},
}

func MockClusterManager() types.ClusterManager {
	clusters := []v2.Cluster{
		clusterOrigin,
	}

	clusterMap := map[string][]v2.Host{
		"o1": []v2.Host{host1, host2},
		"o2": []v2.Host{host3, host4},
	}

	return NewClusterManager(nil, clusters, clusterMap, true, false)
}

func TestNewClusterMngSingle(t *testing.T) {
	mockClusterMnger1 := MockClusterManager().(*clusterManager)
	mockClusterMnger2 := MockClusterManager().(*clusterManager)

	if mockClusterMnger1 != mockClusterMnger2 {
		t.Errorf("error")
	}
	mockClusterMnger1.Destory()
}

func TestGetClusterMngAdapterInstance(t *testing.T) {

	mockClusterMnger := MockClusterManager().(*clusterManager)
	defer mockClusterMnger.Destory()

	tests := []struct {
		name string
		want *clusterManager
	}{
		{
			name: "validtest",
			want: mockClusterMnger,
		},
		{
			name: "validtest2",
			want: mockClusterMnger,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetClusterMngAdapterInstance().clusterMng; got != tt.want {
				t.Errorf("GetClusterMngAdapterInstance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMngAdapter_TriggerClusterAddOrUpdate(t *testing.T) {

	mockClusterMnger := MockClusterManager().(*clusterManager)
	defer mockClusterMnger.Destory()
	mockNewCluster := v2.Cluster{
		Name:        "o1",
		ClusterType: v2.DYNAMIC_CLUSTER,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy:  1,
			DefaultSubset:   map[string]string{"label": "blue"},
			SubsetSelectors: [][]string{{"label"}},
		},
		LbType: v2.LB_RANDOM,
		Hosts:  []v2.Host{host1, host2},
	}

	mockAddedCluster := v2.Cluster{
		Name:        "c1",
		ClusterType: v2.DYNAMIC_CLUSTER,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy:  1,
			DefaultSubset:   map[string]string{"label": "blue"},
			SubsetSelectors: [][]string{{"label"}},
		},
		LbType: v2.LB_RANDOM,
		Hosts:  []v2.Host{host3, host4},
	}

	type fields struct {
		clusterName string
		clusterMng  *clusterManager
	}
	type args struct {
		cluster v2.Cluster
	}

	type clusterArgs struct {
		hostNumber  int
		clusterName string
		err         error
		lbType      types.LoadBalancerType
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantArgs clusterArgs
	}{
		{
			name:   "clusterUpdateTest",
			fields: fields{clusterMng: mockClusterMnger},
			args:   args{cluster: mockNewCluster},
			wantArgs: clusterArgs{
				hostNumber:  2,
				err:         nil,
				clusterName: "o1",
				lbType:      types.Random,
			},
		},
		{
			name:   "clusterAddTest",
			fields: fields{clusterMng: mockClusterMnger},
			args:   args{cluster: mockAddedCluster},
			wantArgs: clusterArgs{
				hostNumber:  0,
				clusterName: "c1",
				lbType:      types.Random,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := &MngAdapter{
				clusterMng: tt.fields.clusterMng,
			}
			if err := ca.TriggerClusterAddOrUpdate(tt.args.cluster); err != tt.wantArgs.err {
				t.Errorf("MngAdapter.TriggerClusterAddOrUpdate() error = %v, wantArgs %v", err, tt.wantArgs)
			}

			if cluster, ok := mockClusterMnger.primaryClusters.Load(tt.wantArgs.clusterName); ok {

				cInMem := cluster.(*primaryCluster).cluster.(*simpleInMemCluster)
				if len(cInMem.hosts) != tt.wantArgs.hostNumber {
					t.Errorf("MngAdapter.update cluster error, want %d hosts, but gotwantErr %v", tt.wantArgs.hostNumber, len(cInMem.hosts))
				}

				if cluster.(*primaryCluster).cluster.Info().LbType() != tt.wantArgs.lbType {
					t.Errorf("MngAdapter.update cluster error")
				}

			} else {
				t.Errorf("MngAdapter.update cluster error, cluster not founf")

			}

		})
	}
}

func TestMngAdapter_TriggerClusterDel(t *testing.T) {
	mockClusterMnger := MockClusterManager().(*clusterManager)
	defer mockClusterMnger.Destory()
	type fields struct {
		clusterMng *clusterManager
	}
	type args struct {
		clusterName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testValid",
			fields: fields{
				clusterMng: mockClusterMnger,
			},
			args: args{
				clusterName: "o1",
			},
			wantErr: false,
		},
		{
			name: "testInvalid",
			fields: fields{
				clusterMng: mockClusterMnger,
			},
			args: args{
				clusterName: "o3",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := &MngAdapter{
				clusterMng: tt.fields.clusterMng,
			}
			if err := ca.TriggerClusterDel(tt.args.clusterName); (err != nil) != tt.wantErr {
				t.Errorf("MngAdapter.TriggerClusterDel() error = %v, wantArgs %v", err, tt.wantErr)
			}

			if _, ok := mockClusterMnger.primaryClusters.Load(tt.args.clusterName); ok {
				t.Errorf("MngAdapter.cluster delete error")
			}

		})
	}
}

func TestMngAdapter_TriggerClusterAndHostsAddOrUpdate(t *testing.T) {
	mockClusterMnger := MockClusterManager().(*clusterManager)
	defer mockClusterMnger.Destory()
	mockNewCluster := v2.Cluster{
		Name:        "o1",
		ClusterType: v2.DYNAMIC_CLUSTER,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy:  1,
			DefaultSubset:   map[string]string{"label": "blue"},
			SubsetSelectors: [][]string{{"label"}},
		},
		LbType: v2.LB_RANDOM,
		Hosts:  []v2.Host{host1},
	}

	mockNewCluster2 := v2.Cluster{
		Name:        "o1",
		ClusterType: v2.DYNAMIC_CLUSTER,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy:  1,
			DefaultSubset:   map[string]string{"label": "blue"},
			SubsetSelectors: [][]string{{"label"}},
		},
		LbType: v2.LB_RANDOM,
		Hosts:  []v2.Host{host1, host2, host3},
	}

	mockNewCluster3 := v2.Cluster{
		Name:        "o1",
		ClusterType: v2.DYNAMIC_CLUSTER,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy:  1,
			DefaultSubset:   map[string]string{"label": "blue"},
			SubsetSelectors: [][]string{{"label"}},
		},
		LbType: v2.LB_RANDOM,
		Hosts:  []v2.Host{},
	}

	type fields struct {
		clusterMng *clusterManager
	}
	type args struct {
		cluster v2.Cluster
		hosts   []v2.Host
	}

	type clusterArgs struct {
		hostNumber    int
		hostName      string
		err           bool
		lbType        types.LoadBalancerType
		clusterType   v2.ClusterType
		clusterNumber int
		c2Exist       bool
	}

	tests := []struct {
		name     string
		fields   fields
		args     args
		argsWant clusterArgs
	}{
		{
			name: "deleteHosts",
			fields: fields{
				clusterMng: mockClusterMnger,
			},
			args: args{
				cluster: mockNewCluster,
				hosts:   []v2.Host{host1},
			},
			argsWant: clusterArgs{
				hostName:   "h1",
				hostNumber: 1,
				err:        true,
				lbType:     types.Random,
			},
		},
		{
			name: "addHosts",
			fields: fields{
				clusterMng: mockClusterMnger,
			},
			args: args{
				cluster: mockNewCluster2,
				hosts:   []v2.Host{host1, host2, host3},
			},
			argsWant: clusterArgs{
				hostNumber: 3,
				err:        true,
				lbType:     types.Random,
			},
		},

		{
			name: "clearHosts",
			fields: fields{
				clusterMng: mockClusterMnger,
			},
			args: args{
				cluster: mockNewCluster3,
				hosts:   []v2.Host{},
			},
			argsWant: clusterArgs{
				hostNumber: 0,
				err:        true,
				lbType:     types.Random,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ca := &MngAdapter{
				clusterMng: tt.fields.clusterMng,
			}

			if err := ca.TriggerClusterAndHostsAddOrUpdate(tt.args.cluster, tt.args.hosts); (err == nil) != tt.argsWant.err {
				t.Errorf("MngAdapter.TriggerClusterAndHostsAddOrUpdate() error = %v, wantArgs %v", err, nil)
			}

			if cluster, ok := mockClusterMnger.primaryClusters.Load("o1"); ok {

				if cluster.(*primaryCluster).cluster.Info().LbType() != tt.argsWant.lbType {
					t.Errorf("MngAdapter.update cluster error")
				}

				cInMem := cluster.(*primaryCluster).cluster.(*simpleInMemCluster)
				if len(cInMem.hosts) != tt.argsWant.hostNumber {
					t.Errorf("MngAdapter.update cluster and host error, want %d hosts, but got = %d", tt.argsWant.hostNumber, len(cInMem.hosts))
				}

				if tt.name == "deleteHosts" {
					if cInMem.hosts[0].Hostname() != tt.argsWant.hostName {
						t.Errorf("MngAdapter.update cluster and host error, want host = %s, but got %v", tt.argsWant.hostName, cInMem.hosts[0].Hostname())
					}
				}
			} else {
				t.Errorf("MngAdapter.TriggerClusterAndHostsAddOrUpdate() ")
			}

		})
	}
}

func TestMngAdapter_TriggerClusterHostUpdate(t *testing.T) {
	mockClusterMnger := MockClusterManager().(*clusterManager)
	defer mockClusterMnger.Destory()

	type fields struct {
		clusterMng *clusterManager
	}
	type args struct {
		clusterName string
		hosts       []v2.Host
	}

	type clusterArgs struct {
		hostNumber    int
		hostName      string
		err           bool
		lbType        types.LoadBalancerType
		clusterType   v2.ClusterType
		clusterNumber int
		c2Exist       bool
	}

	tests := []struct {
		name     string
		fields   fields
		args     args
		argsWant clusterArgs
	}{
		{
			name: "deleteHosts",
			fields: fields{
				clusterMng: mockClusterMnger,
			},
			args: args{
				clusterName: "o1",
				hosts:       []v2.Host{host1},
			},
			argsWant: clusterArgs{
				hostName:   "h1",
				hostNumber: 1,
				err:        true,
				lbType:     types.RoundRobin,
			},
		},
		{
			name: "addHosts",
			fields: fields{
				clusterMng: mockClusterMnger,
			},
			args: args{
				clusterName: "o1",
				hosts:       []v2.Host{host1, host2, host3},
			},
			argsWant: clusterArgs{
				hostNumber: 3,
				err:        true,
				lbType:     types.RoundRobin,
			},
		},

		{
			name: "clearHosts",
			fields: fields{
				clusterMng: mockClusterMnger,
			},
			args: args{
				clusterName: "o1",
				hosts:       []v2.Host{},
			},
			argsWant: clusterArgs{
				hostNumber: 0,
				err:        true,
				lbType:     types.RoundRobin,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := &MngAdapter{
				clusterMng: tt.fields.clusterMng,
			}

			if err := ca.TriggerClusterHostUpdate(tt.args.clusterName, tt.args.hosts); (err == nil) != tt.argsWant.err {
				t.Errorf("MngAdapter.TriggerClusterAndHostsAddOrUpdate() error = %v, wantArgs %v", err, nil)
			}

			if cluster, ok := mockClusterMnger.primaryClusters.Load("o1"); ok {

				if cluster.(*primaryCluster).cluster.Info().LbType() != tt.argsWant.lbType {
					t.Errorf("MngAdapter.update cluster error")
				}

				cInMem := cluster.(*primaryCluster).cluster.(*simpleInMemCluster)
				if len(cInMem.hosts) != tt.argsWant.hostNumber {
					t.Errorf("MngAdapter.update cluster and host error, want %d hosts, but got = %d", tt.argsWant.hostNumber, len(cInMem.hosts))
				}

				if tt.name == "deleteHosts" {
					if cInMem.hosts[0].Hostname() != tt.argsWant.hostName {
						t.Errorf("MngAdapter.update cluster and host error, want host = %s, but got %v", tt.argsWant.hostName, cInMem.hosts[0].Hostname())
					}
				}
			} else {
				t.Errorf("MngAdapter.TriggerClusterAndHostsAddOrUpdate() ")
			}
		})
	}
}

func TestMngAdapter_TriggerHostDel(t *testing.T) {
	mockClusterMnger := MockClusterManager().(*clusterManager)
	defer mockClusterMnger.Destory()

	type fields struct {
		clusterMng *clusterManager
	}
	type args struct {
		clusterName string
		hostAddress string
	}
	type argsWant struct {
		wantErr     bool
		hostNumber  int
		hostAddress string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		argsWant argsWant
	}{
		{
			name: "validDel",
			fields: fields{
				clusterMng: mockClusterMnger,
			},
			args: args{
				clusterName: "o1",
				hostAddress: "127.0.0.1",
			},
			argsWant: argsWant{
				wantErr:     false,
				hostNumber:  1,
				hostAddress: "127.0.0.2",
			},
		},
		{
			name: "allDel",
			fields: fields{
				clusterMng: mockClusterMnger,
			},
			args: args{
				clusterName: "o1",
				hostAddress: "127.0.0.2",
			},
			argsWant: argsWant{
				wantErr:     false,
				hostNumber:  0,
				hostAddress: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := &MngAdapter{
				clusterMng: tt.fields.clusterMng,
			}
			if err := ca.TriggerHostDel(tt.args.clusterName, tt.args.hostAddress); (err != nil) != tt.argsWant.wantErr {
				t.Errorf("MngAdapter.TriggerHostDel() error = %v, wantErr %v", err, tt.argsWant.wantErr)
			}

			if cluster, ok := mockClusterMnger.primaryClusters.Load(tt.args.clusterName); ok {

				cInMem := cluster.(*primaryCluster).cluster.(*simpleInMemCluster)
				if len(cInMem.hosts) != tt.argsWant.hostNumber {
					t.Errorf("MngAdapter.update cluster and host error, want %d hosts, but got = %d", tt.argsWant.hostNumber, len(cInMem.hosts))
				}

				if len(cInMem.hosts) > 0 && cInMem.hosts[0].AddressString() != tt.argsWant.hostAddress {
					t.Errorf("MngAdapter.update cluster and host error, want host = %s, but got %v", tt.argsWant.hostAddress, cInMem.hosts[0].AddressString())
				}

			} else {
				t.Errorf("MngAdapter.TriggerClusterAndHostsAddOrUpdate() ")
			}
		})
	}
}
