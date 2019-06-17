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

	"sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/types"
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
		MetaData: v2.Metadata{
			"version": "1.0.0",
			"zone":    "a",
		},
	}
	host2 := v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1:10001",
		},
		MetaData: v2.Metadata{
			"version": "2.0.0",
			"zone":    "a",
		},
	}
	clusterMangerInstance.Destroy() // Destroy for test
	return NewClusterManagerSingleton([]v2.Cluster{clusterConfig}, map[string][]v2.Host{
		"test1": []v2.Host{host1, host2},
	})
}

func TestClusterManagerFromConfig(t *testing.T) {
	// create simple example config
	_createClusterManager()
	// use get for test
	snap := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	defer GetClusterMngAdapterInstance().PutClusterSnapshot(snap)
	if snap == nil {
		t.Fatal("can not get snapshot")
	}
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
	if snap.IsExistsHosts(mockLb5.MetadataMatchCriteria()) {
		t.Fatal("host not exists is not expected")
	}
}

func TestClusterManagerDynamic(t *testing.T) {
	_createClusterManager()
	if GetClusterMngAdapterInstance().ClusterExist("test2") {
		t.Fatal("exists unexpected cluster")
	}
	// Add cluster
	if err := GetClusterMngAdapterInstance().AddOrUpdatePrimaryCluster(v2.Cluster{
		Name:   "test2",
		LbType: v2.LB_RANDOM,
	}); err != nil {
		t.Fatal("update cluster failed: ", err)
	}
	if !(GetClusterMngAdapterInstance().ClusterExist("test1") && GetClusterMngAdapterInstance().ClusterExist("test2")) {
		t.Fatal("cluster add failed")
	}
	// Add Host
	if err := GetClusterMngAdapterInstance().UpdateClusterHosts("test2", []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1:20000",
			},
			MetaData: v2.Metadata{
				"version": "1.0.0",
			},
		},
	}); err != nil {
		t.Fatal("update cluster hosts failed:", err)
	}
	// Update cluster
	if err := GetClusterMngAdapterInstance().AddOrUpdatePrimaryCluster(v2.Cluster{
		Name:   "test2",
		LbType: v2.LB_RANDOM,
		LBSubSetConfig: v2.LBSubsetConfig{
			SubsetSelectors: [][]string{
				[]string{"version"},
				[]string{"version", "zone"},
			},
		},
	}); err != nil {
		t.Fatal("update cluster failed: ", err)
	}
	snap := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test2")
	defer GetClusterMngAdapterInstance().PutClusterSnapshot(snap)
	if host := snap.LoadBalancer().ChooseHost(newMockLbContext((map[string]string{
		"version": "1.0.0",
	}))); host == nil {
		t.Fatal("cluster updated, but lb result is not expected")
	}
	if host := snap.LoadBalancer().ChooseHost(newMockLbContext((map[string]string{
		"version": "2.0.0",
		"zone":    "b",
	}))); host != nil {
		t.Fatal("get a host, but expected not")
	}
	// Append Host
	if err := GetClusterMngAdapterInstance().AppendClusterHosts("test2", []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1:20001",
			},
			MetaData: v2.Metadata{
				"version": "2.0.0",
				"zone":    "b",
			},
		},
	}); err != nil {
		t.Fatal("Append Host Failed", err)
	}
	if host := snap.LoadBalancer().ChooseHost(newMockLbContext(map[string]string{
		"version": "2.0.0",
		"zone":    "b",
	})); host == nil {
		t.Fatal("cluster updated, but lb result is not expected")
	}
	// RemoveHost
	if err := GetClusterMngAdapterInstance().RemoveClusterHosts("test2", []string{"127.0.0.1:20001"}); err != nil {
		t.Fatal("Remove Host Failed", err)
	}
	if host := snap.LoadBalancer().ChooseHost(newMockLbContext(map[string]string{
		"version": "2.0.0",
		"zone":    "b",
	})); host != nil {
		t.Fatal("host removed, but still can be found")
	}
	// RemoveCluster
	if err := GetClusterMngAdapterInstance().RemovePrimaryCluster("test1", "test2"); err != nil {
		t.Fatal("remove clusters failed:", err)
	}
	// expected delete all
	count := 0
	clusterMangerInstance.primaryClusterMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	if count != 0 {
		t.Fatal("cluster still have cluster, but expected to remove all of it")
	}

}

func TestConnPoolForCluster(t *testing.T) {
	_createClusterManager()
	snap := GetClusterMngAdapterInstance().GetClusterSnapshot(nil, "test1")
	defer GetClusterMngAdapterInstance().PutClusterSnapshot(snap)
	connPool := GetClusterMngAdapterInstance().ConnPoolForCluster(newMockLbContext(nil), snap, mockProtocol)
	if connPool == nil {
		t.Fatal("get conn pool failed")
	}
}
