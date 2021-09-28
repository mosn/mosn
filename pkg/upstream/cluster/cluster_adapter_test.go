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
	"testing"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

// we call cluster manager by cluster adapter

func _createClusterManager() types.ClusterManager {
	clusterConfig := v2.Cluster{
		Name:   "test1",
		LbType: v2.LB_RANDOM,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy: 1, // AnyEndPoint
			SubsetSelectors: [][]string{
				[]string{"version"},
				[]string{"version", "zone"},
			},
		},
	}
	host1 := v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1:10000",
		},
		MetaData: api.Metadata{
			"version": "1.0.0",
			"zone":    "a",
		},
	}
	host2 := v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1:10001",
		},
		MetaData: api.Metadata{
			"version": "2.0.0",
			"zone":    "a",
		},
	}
	clusterManagerInstance.Destroy() // Destroy for test
	return NewClusterManagerSingleton([]v2.Cluster{clusterConfig}, map[string][]v2.Host{
		"test1": []v2.Host{host1, host2},
	}, nil)
}

func TestClusterManagerFromConfig(t *testing.T) {
	// create simple example config
	_createClusterManager()
	// use get for test
	snap := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	// check hosts exists
	// check subset exists
	mockLb1 := newMockLbContext(map[string]string{
		"version": "1.0.0",
	})
	mockLb2 := newMockLbContext(map[string]string{
		"version": "2.0.0",
	})
	mockLb3 := newMockLbContext(map[string]string{
		"version": "1.0.0",
		"zone":    "a",
	})
	mockLb4 := newMockLbContext(map[string]string{
		"version": "2.0.0",
		"zone":    "a",
	})
	if !(snap.IsExistsHosts(nil) &&
		snap.IsExistsHosts(mockLb1.MetadataMatchCriteria()) &&
		snap.IsExistsHosts(mockLb2.MetadataMatchCriteria()) &&
		snap.IsExistsHosts(mockLb3.MetadataMatchCriteria()) &&
		snap.IsExistsHosts(mockLb4.MetadataMatchCriteria())) {
		t.Fatal("host exists is not expected")
	}
	// check subset not exists
	mockLb5 := newMockLbContext(map[string]string{
		"zone": "a",
	})
	if !snap.IsExistsHosts(mockLb5.MetadataMatchCriteria()) {
		t.Fatal("host not exists is not expected")
	}
}

func TestClusterManagerAddCluster(t *testing.T) {
	_createClusterManager()
	if GetClusterMngAdapterInstance().ClusterExist("test2") {
		t.Fatal("exists unexpected cluster")
	}
	// Add cluster
	if err := GetClusterMngAdapterInstance().TriggerClusterAddOrUpdate(v2.Cluster{
		Name:   "test2",
		LbType: v2.LB_RANDOM,
	}); err != nil {
		t.Fatal("update cluster failed: ", err)
	}
	if !(GetClusterMngAdapterInstance().ClusterExist("test1") && GetClusterMngAdapterInstance().ClusterExist("test2")) {
		t.Fatal("cluster add failed")
	}
}

func TestClusterManagerUpdateCluster(t *testing.T) {
	_createClusterManager()
	if !GetClusterMngAdapterInstance().ClusterExist("test1") {
		t.Fatal("not exists expected cluster")
	}

	var maxc uint32 = 8
	clusterConfig := v2.Cluster{
		Name:   "test1",
		LbType: v2.LB_RANDOM,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy: 1, // AnyEndPoint
			SubsetSelectors: [][]string{
				[]string{"version"},
				[]string{"version", "zone"},
			},
		},
		CirBreThresholds: v2.CircuitBreakers{
			[]v2.Thresholds{
				{
					MaxConnections: maxc,
				},
			}},
	}
	// Update cluster info
	if err := GetClusterMngAdapterInstance().TriggerClusterAddOrUpdate(
		clusterConfig); err != nil {
		t.Fatal("update cluster failed: ", err)
	}

	snapshot := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	rm := snapshot.ClusterInfo().ResourceManager()
	if rm.Connections().Max() != uint64(maxc) {
		t.Fatal("ResourceManager update failed")
	}

	if !GetClusterMngAdapterInstance().ClusterExist("test1") {
		t.Fatal("cluster add failed")
	}
	mockLbCtx := newMockLbContext((map[string]string{
		"zone":    "a",
		"version": "1.0.0"}))

	pool, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snapshot, mockProtocol)

	if pool.Host().ClusterInfo().ResourceManager().Connections().Max() != uint64(maxc) {
		t.Fatal("update cluster resource failed")
	}

	var maxc1 uint32 = 9
	clusterConfig = v2.Cluster{
		Name:   "test1",
		LbType: v2.LB_RANDOM,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy: 1, // AnyEndPoint
			SubsetSelectors: [][]string{
				[]string{"version"},
				[]string{"version", "zone"},
			},
		},
		CirBreThresholds: v2.CircuitBreakers{
			[]v2.Thresholds{
				{
					MaxConnections: maxc1,
				},
			}},
	}

	// test cluster info update
	if err := GetClusterMngAdapterInstance().TriggerClusterAddOrUpdate(
		clusterConfig); err != nil {
		t.Fatal("update cluster failed: ", err)
	}

	pool, _ = GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snapshot, mockProtocol)
	if pool.Host().ClusterInfo().ResourceManager().Connections().Max() != uint64(maxc1) {
		t.Fatal("update cluster resource failed")
	}

	// test cluster host update
	host1 := v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1:10002",
		},
		MetaData: api.Metadata{
			"version": "1.0.0",
			"zone":    "a",
		},
	}
	host2 := v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1:10003",
		},
		MetaData: api.Metadata{
			"version": "2.0.0",
			"zone":    "a",
		},
	}

	if err := GetClusterMngAdapterInstance().TriggerClusterHostUpdate(
		"test1", []v2.Host{host1, host2}); err != nil {
		t.Fatal("update cluster failed: ", err)
	}

	snapshot = GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	pool, _ = GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snapshot, mockProtocol)
	if pool.Host().ClusterInfo().ResourceManager().Connections().Max() != uint64(maxc1) {
		t.Fatal("update cluster resource failed")
	}
	if pool.Host().SupportTLS() {
		t.Fatal("pool should not support tls")
	}
	// test cluster update tls context, the host tls context is updated too
	GetClusterMngAdapterInstance().AddOrUpdatePrimaryCluster(v2.Cluster{
		Name:   "test1",
		LbType: v2.LB_RANDOM,
		TLS: v2.TLSConfig{
			Status:       true,
			InsecureSkip: true,
		},
	})
	// pool keeps snapshot
	snapshot = GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	pool, _ = GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snapshot, mockProtocol)
	if !pool.Host().SupportTLS() {
		t.Fatal("pool should support tls")
	}

}

func TestClusterManagerRemoveCluster(t *testing.T) {
	_createClusterManager()
	if err := GetClusterMngAdapterInstance().TriggerClusterDel("test1", "test2"); err == nil {
		t.Fatal("can not remove cluster not exists")
	}
	if !GetClusterMngAdapterInstance().ClusterExist("test1") {
		t.Fatal("cluster should still exists, but not")
	}
	GetClusterMngAdapterInstance().TriggerClusterDel("test1")
	if GetClusterMngAdapterInstance().ClusterExist("test1") {
		t.Fatal("cluster should be deleted, but not")
	}
}

// TestClusterManagerUpdateClusterSelectors update cluster configs (selectors)
// It makes a new subset clusters, keeps the hosts
func TestClusterManagerUpdateClusterSelectors(t *testing.T) {
	_createClusterManager()
	// UpdateCluster
	if err := GetClusterMngAdapterInstance().AddOrUpdatePrimaryCluster(v2.Cluster{
		Name:   "test1",
		LbType: v2.LB_RANDOM,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy: 0, // No Fallback
			SubsetSelectors: [][]string{
				[]string{"version"},
			},
		},
	}); err != nil {
		t.Fatal("update cluster failed:", err)
	}
	snap := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	if host := snap.LoadBalancer().ChooseHost(newMockLbContext((map[string]string{
		"zone":    "a",
		"version": "1.0.0",
	}))); host != nil {
		t.Fatal("expected host not found, but not")
	}
}

// TestClusterUpdateHostsWithSnapshot updates a new hosts in cluster
// the new snapshot will get the new hosts, but the old snapshot still keeps the old hosts
func TestClusterUpdateHostsWithSnapshot(t *testing.T) {
	_createClusterManager()
	mockLb1 := newMockLbContext(map[string]string{
		"version": "3.0.0",
	})
	mockLb2 := newMockLbContext(map[string]string{
		"version": "1.0.0",
		"zone":    "a",
	})
	mockLb3 := newMockLbContext(map[string]string{
		"version": "1.0.0",
		"zone":    "b",
	})
	oldSnap := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	// Update Hosts
	GetClusterMngAdapterInstance().TriggerClusterHostUpdate("test1", []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1:10000",
			},
			MetaData: api.Metadata{
				"version": "3.0.0",
			},
		},
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1:10002",
			},
			MetaData: api.Metadata{
				"version": "1.0.0",
				"zone":    "b",
			},
		},
	})
	newSnap := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	if !(oldSnap.IsExistsHosts(mockLb1.MetadataMatchCriteria()) &&
		oldSnap.IsExistsHosts(mockLb2.MetadataMatchCriteria()) &&
		oldSnap.IsExistsHosts(mockLb3.MetadataMatchCriteria())) {
		t.Fatal("old snapshot is changed")
	}
	if !(newSnap.IsExistsHosts(mockLb1.MetadataMatchCriteria()) &&
		newSnap.IsExistsHosts(mockLb2.MetadataMatchCriteria()) &&
		newSnap.IsExistsHosts(mockLb3.MetadataMatchCriteria())) {
		t.Fatal("new snapshot is not expected")
	}
}

func TestClusterAppendHostWithSnapshot(t *testing.T) {
	_createClusterManager()
	oldSnap := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	GetClusterMngAdapterInstance().TriggerHostAppend("test1", []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1:10002",
			},
		},
	})
	newSnap := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	if !(oldSnap.HostSet().Size() == 2 && newSnap.HostSet().Size() == 3) {
		t.Fatalf("append hosts snapshot check failed, old: %d, new: %d ", oldSnap.HostSet().Size(), newSnap.HostSet().Size())
	}
}

func TestClusterRemoveHostWithSnapshot(t *testing.T) {
	_createClusterManager()
	oldSnap := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	GetClusterMngAdapterInstance().TriggerHostDel("test1", []string{"127.0.0.1:10001"})
	newSnap := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	if !(oldSnap.HostSet().Size() == 2 && newSnap.HostSet().Size() == 1) {
		t.Fatal("remove hosts snapshot check failed")
	}
}

func TestConnPoolForCluster(t *testing.T) {
	_createClusterManager()
	snap := GetClusterMngAdapterInstance().GetClusterSnapshot(nil, "test1")
	connPool, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap, mockProtocol)
	if connPool == nil {
		t.Fatal("get conn pool failed")
	}
}

func TestConnPoolUpdateTLS(t *testing.T) {
	testStateReset()
	defer testStateReset()
	clusterConfig := v2.Cluster{
		Name:   "test1",
		LbType: v2.LB_RANDOM,
		TLS: v2.TLSConfig{
			Status:       true,
			InsecureSkip: true,
		},
	}
	host := v2.Host{
		HostConfig: v2.HostConfig{
			Address:    "127.0.0.1:10000",
			TLSDisable: true,
		},
	}
	clusterManagerInstance.Destroy() // Destroy for test
	NewClusterManagerSingleton([]v2.Cluster{clusterConfig}, map[string][]v2.Host{
		"test1": []v2.Host{host},
	}, nil)
	snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(nil, "test1")
	connPool1, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap1, mockProtocol)
	// TLSDisable always returns disable hash value
	if !connPool1.TLSHashValue().Equal(disableTLSHashValue) {
		t.Fatal("conn pool support tls")
	}
	if err := GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1:10000",
			},
		},
	}); err != nil {
		t.Fatalf("update cluster hosts failed, %v", err)
	}
	snap2 := GetClusterMngAdapterInstance().GetClusterSnapshot(nil, "test1")
	connPool2, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap2, mockProtocol)
	cp2Hash := connPool2.TLSHashValue()
	if cp2Hash.Equal(disableTLSHashValue) || cp2Hash.Equal(nil) || cp2Hash.Equal(clientSideDisableHashValue) {
		t.Fatal("conn pool does not support tls")
	}
	// disable tls, connpool should will be changed
	DisableClientSideTLS()
	connPool3, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap2, mockProtocol)
	// connpool should be changed, but old connpool should not be effected
	if connPool2.TLSHashValue().Equal(disableTLSHashValue) || connPool2.TLSHashValue().Equal(nil) || connPool2.TLSHashValue().Equal(clientSideDisableHashValue) {
		t.Fatal("old conn pool does not support tls")
	}
	if !connPool3.TLSHashValue().Equal(clientSideDisableHashValue) {
		t.Fatal("conn pool support tls")
	}
}

func TestClusterManagerTLSUpdateTLS(t *testing.T) {
	testStateReset()
	defer testStateReset()
	clusterConfig := v2.Cluster{
		Name:              "test1",
		LbType:            v2.LB_RANDOM,
		ClusterManagerTLS: true,
	}
	host := v2.Host{
		HostConfig: v2.HostConfig{
			Address:    "127.0.0.1:10000",
			TLSDisable: true,
		},
	}
	clusterManagerInstance.Destroy() // Destroy for test
	NewClusterManagerSingleton([]v2.Cluster{clusterConfig}, map[string][]v2.Host{
		"test1": []v2.Host{host},
	}, nil)
	snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(nil, "test1")
	connPool1, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap1, mockProtocol)
	// should returns disableTLSHashValue because TLSDisable is true
	if !connPool1.TLSHashValue().Equal(disableTLSHashValue) {
		t.Fatal("conn pool hash value should be disableTLSHashValue")
	}
	// disable tls, keeps disableTLSHashValue not changed
	DisableClientSideTLS()
	connPool2, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap1, mockProtocol)
	if !connPool2.TLSHashValue().Equal(connPool1.TLSHashValue()) {
		t.Fatalf("connpool should not be changed, hash value shoule be keeped as disableTLSHashValue")
	}
	//
	EnableClientSideTLS()
	connPool3, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap1, mockProtocol)
	if !connPool3.TLSHashValue().Equal(connPool2.TLSHashValue()) {
		t.Fatalf("connpool should not be changed")
	}
	// Update to support TLS
	GetClusterMngAdapterInstance().UpdateTLSManager(&v2.TLSConfig{
		Status:       true,
		InsecureSkip: true,
	})
	if err := GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1:10000",
			},
		},
	}); err != nil {
		t.Fatalf("update cluster hosts failed, %v", err)
	}
	snap2 := GetClusterMngAdapterInstance().GetClusterSnapshot(nil, "test1")
	connPool4, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap2, mockProtocol)
	// connpool snapshot keeps disable
	if !connPool1.TLSHashValue().Equal(disableTLSHashValue) {
		t.Fatal("connpool snapshot keeps nothing to change")
	}
	if connPool4.TLSHashValue().Equal(nil) || connPool4.TLSHashValue().Equal(disableTLSHashValue) || connPool4.TLSHashValue().Equal(clientSideDisableHashValue) {
		t.Fatal("conn pool does not support tls")
	}
	// Update Same Config, no effects
	GetClusterMngAdapterInstance().UpdateTLSManager(&v2.TLSConfig{
		Status:       true,
		InsecureSkip: true,
	})
	connPool5, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap2, mockProtocol)
	if !connPool5.TLSHashValue().Equal(connPool4.TLSHashValue()) {
		t.Fatal("conn pool changed, but no config changed")
	}
	// disable tls, connpool should will be changed
	DisableClientSideTLS()
	connPool6, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap2, mockProtocol)
	if !connPool6.TLSHashValue().Equal(clientSideDisableHashValue) {
		t.Fatalf("conn pool hash value should be nil")
	}
	// update tls config when disable tls
	if err := GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address:    "127.0.0.1:10000",
				TLSDisable: true,
			},
		},
	}); err != nil {
		t.Fatalf("update cluster hosts failed, %v", err)
	}
	snap3 := GetClusterMngAdapterInstance().GetClusterSnapshot(nil, "test1")
	connPool7, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap3, mockProtocol)
	// connpool should be changed.
	if !connPool7.TLSHashValue().Equal(disableTLSHashValue) {
		t.Fatalf("conn pool should be changed to disableTLSHashValue")
	}
	EnableClientSideTLS()
	connPool8, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap3, mockProtocol)
	if !connPool8.TLSHashValue().Equal(connPool7.TLSHashValue()) {
		t.Fatal("conn pool should not be changed")
	}
	if !connPool8.TLSHashValue().Equal(disableTLSHashValue) {
		t.Fatal("should be disabled hash value")
	}
}
