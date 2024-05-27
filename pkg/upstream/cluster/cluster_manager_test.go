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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

func TestClusterUpdateAndHosts(t *testing.T) {
	_createClusterManager()
	config := v2.Cluster{
		Name:        "test",
		ClusterType: v2.SIMPLE_CLUSTER,
		CirBreThresholds: v2.CircuitBreakers{
			Thresholds: []v2.Thresholds{
				{
					MaxConnections: 100,
				},
			},
		},
	}
	clusterManagerInstance.AddOrUpdateClusterAndHost(config, []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1:10000",
			},
		},
	})
	snap1 := clusterManagerInstance.GetClusterSnapshot(context.Background(), "test")
	snap1.ClusterInfo().ResourceManager().Connections().Increase()
	require.Equal(t, uint64(100), snap1.ClusterInfo().ResourceManager().Connections().Max())
	require.Equal(t, 1, snap1.HostNum(nil))

	oldLastHealthCheckPassTimes := make(map[string]time.Time)
	snap1.HostSet().Range(func(host types.Host) bool {
		oldLastHealthCheckPassTimes[host.AddressString()] = host.LastHealthCheckPassTime()
		return true
	})

	newConfig := v2.Cluster{
		Name:        "test",
		ClusterType: v2.SIMPLE_CLUSTER,
		CirBreThresholds: v2.CircuitBreakers{
			Thresholds: []v2.Thresholds{
				{
					MaxConnections: 20,
				},
			},
		},
	}
	clusterManagerInstance.AddOrUpdatePrimaryCluster(newConfig)
	snap2 := clusterManagerInstance.GetClusterSnapshot(context.Background(), "test")
	// hosts will be inheritted
	// resource manager config will be updated
	require.Equal(t, 1, snap2.HostNum(nil))
	require.Equal(t, int64(1), snap2.ClusterInfo().ResourceManager().Connections().Cur())
	require.Equal(t, uint64(20), snap2.ClusterInfo().ResourceManager().Connections().Max())

	// lastHealthCheckPassTime of hosts are transferred
	snap2.HostSet().Range(func(host types.Host) bool {
		st := oldLastHealthCheckPassTimes[host.AddressString()]
		require.Equal(t, st, host.LastHealthCheckPassTime())
		return true
	})
}

func TestClusterManager_ConnPoolForCluster(t *testing.T) {

	h := v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1:10000",
		},
	}

	t.Run("cluster manager clusterPoolEnable is true, cluster false", func(t *testing.T) {
		_createClusterManagerWithConfig(&v2.ClusterManagerConfig{
			ClusterManagerConfigJson: v2.ClusterManagerConfigJson{
				ClusterPoolEnable: true,
			},
		})
		GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{h})
		snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
		mockLbCtx := newMockLbContext(nil)
		p, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		if p == nil {
			t.Fatal("get conn pool failed")
		}
		require.False(t, globalPoolExists(h.Address))
		require.True(t, clusterPoolExists(h.Address, snap1))
	})

	t.Run("cluster manager clusterPoolEnable is false, cluster false", func(t *testing.T) {
		_createClusterManager()
		GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{h})
		snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
		mockLbCtx := newMockLbContext(nil)
		p, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		if p == nil {
			t.Fatal("get conn pool failed")
		}
		require.True(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
	})

	t.Run("cluster manager clusterPoolEnable is false, cluster true", func(t *testing.T) {
		_createClusterManager()
		GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{h})
		clusterConfig2 := v2.Cluster{
			Name:              "test2",
			LbType:            v2.LB_RANDOM,
			LBSubSetConfig:    v2.LBSubsetConfig{},
			ClusterPoolEnable: true,
		}
		h2 := h
		h2.Address = "127.0.0.1:10001"
		GetClusterMngAdapterInstance().AddOrUpdateClusterAndHost(clusterConfig2, []v2.Host{h2})
		snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
		snap2 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test2")
		mockLbCtx := newMockLbContext(nil)
		p1, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		p2, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap2, mockProtocol)
		if p1 == nil || p2 == nil {
			t.Fatal("get conn pool failed")
		}
		// cluster1 defaultPool, cluster2 clusterPool
		require.True(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		require.False(t, globalPoolExists(h2.Address))
		require.True(t, clusterPoolExists(h2.Address, snap2))
	})

	t.Run("cluster manager clusterPoolEnable is false, cluster true", func(t *testing.T) {
		_createClusterManager()
		GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{h})
		clusterConfig2 := v2.Cluster{
			Name:           "test2",
			LbType:         v2.LB_RANDOM,
			LBSubSetConfig: v2.LBSubsetConfig{},
		}
		h2 := h
		h2.Address = "127.0.0.1:10001"
		GetClusterMngAdapterInstance().AddOrUpdateClusterAndHost(clusterConfig2, []v2.Host{h2})
		snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
		snap2 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test2")
		mockLbCtx := newMockLbContext(nil)
		p1, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		p2, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap2, mockProtocol)
		if p1 == nil || p2 == nil {
			t.Fatal("get conn pool failed")
		}
		// cluster1 defaultPool, cluster2 defaultPool
		require.True(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		require.True(t, globalPoolExists(h2.Address))
		require.False(t, clusterPoolExists(h2.Address, snap2))
	})

}

func globalPoolExists(addr string) bool {
	globalPoolMap, _ := GetClusterMngAdapterInstance().ClusterManager.(*clusterManagerSingleton).protocolConnPool.globalPool.Load(mockProtocol)
	_, ok := globalPoolMap.(*sync.Map).Load(addr)
	return ok
}

func clusterPoolExists(addr string, snap types.ClusterSnapshot) bool {
	clusterPoolMap, _ := GetClusterMngAdapterInstance().ClusterManager.(*clusterManagerSingleton).protocolConnPool.clusterPool.Load(mockProtocol)
	connectionPool, clusterExists := clusterPoolMap.(*sync.Map).Load(snap.ClusterInfo().Name())
	if !clusterExists {
		return false
	}
	_, hasClusterPool := connectionPool.(*sync.Map).Load(addr)
	return hasClusterPool
}

func TestClusterManager_ShutdownConnectionPool(t *testing.T) {
	h := v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1:10000",
		},
	}
	t.Run("cluster manager clusterPoolEnable is true, cluster false", func(t *testing.T) {
		_createClusterManagerWithConfig(&v2.ClusterManagerConfig{
			ClusterManagerConfigJson: v2.ClusterManagerConfigJson{
				ClusterPoolEnable: true,
			},
		})
		GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{h})
		snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
		mockLbCtx := newMockLbContext(nil)
		p, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		if p == nil {
			t.Fatal("get conn pool failed")
		}
		require.False(t, globalPoolExists(h.Address))
		require.True(t, clusterPoolExists(h.Address, snap1))
		clusterManagerInstance.ShutdownConnectionPool(mockProtocol, h.Address)
		require.False(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))

		p, _ = GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		if p == nil {
			t.Fatal("get conn pool failed")
		}
		require.False(t, globalPoolExists(h.Address))
		require.True(t, clusterPoolExists(h.Address, snap1))
		clusterManagerInstance.ShutdownConnectionPool("", h.Address)
		require.False(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
	})

	t.Run("cluster manager clusterPoolEnable is false, cluster false", func(t *testing.T) {
		_createClusterManager()
		GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{h})
		snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
		mockLbCtx := newMockLbContext(nil)
		p, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		if p == nil {
			t.Fatal("get conn pool failed")
		}
		require.True(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		clusterManagerInstance.ShutdownConnectionPool(mockProtocol, h.Address)
		require.False(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))

		p, _ = GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		if p == nil {
			t.Fatal("get conn pool failed")
		}
		require.True(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		clusterManagerInstance.ShutdownConnectionPool("", h.Address)
		require.False(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
	})

	t.Run("cluster manager clusterPoolEnable is false, cluster true", func(t *testing.T) {
		_createClusterManager()
		GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{h})
		clusterConfig2 := v2.Cluster{
			Name:              "test2",
			LbType:            v2.LB_RANDOM,
			LBSubSetConfig:    v2.LBSubsetConfig{},
			ClusterPoolEnable: true,
		}
		h2 := h
		GetClusterMngAdapterInstance().AddOrUpdateClusterAndHost(clusterConfig2, []v2.Host{h2})
		snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
		snap2 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test2")
		mockLbCtx := newMockLbContext(nil)
		p1, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		p2, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap2, mockProtocol)
		if p1 == nil || p2 == nil {
			t.Fatal("get conn pool failed")
		}
		// cluster1 defaultPool, cluster2 clusterPool
		require.True(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		require.True(t, globalPoolExists(h2.Address)) // h1.addr == h2.addr
		require.True(t, clusterPoolExists(h2.Address, snap2))
		clusterManagerInstance.ShutdownConnectionPool(mockProtocol, h.Address)
		require.False(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		require.False(t, globalPoolExists(h2.Address))
		require.False(t, clusterPoolExists(h2.Address, snap2))

		p1, _ = GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		p2, _ = GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap2, mockProtocol)
		if p1 == nil || p2 == nil {
			t.Fatal("get conn pool failed")
		}
		// cluster1 defaultPool, cluster2 clusterPool
		require.True(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		require.True(t, globalPoolExists(h2.Address)) // h1.addr == h2.addr
		require.True(t, clusterPoolExists(h2.Address, snap2))
		clusterManagerInstance.ShutdownConnectionPool("", h.Address)
		require.False(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		require.False(t, globalPoolExists(h2.Address))
		require.False(t, clusterPoolExists(h2.Address, snap2))
	})

	t.Run("cluster manager clusterPoolEnable is false, cluster true", func(t *testing.T) {
		_createClusterManager()
		GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{h})
		clusterConfig2 := v2.Cluster{
			Name:           "test2",
			LbType:         v2.LB_RANDOM,
			LBSubSetConfig: v2.LBSubsetConfig{},
		}
		h2 := h
		GetClusterMngAdapterInstance().AddOrUpdateClusterAndHost(clusterConfig2, []v2.Host{h2})
		snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
		snap2 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test2")
		mockLbCtx := newMockLbContext(nil)
		p1, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		p2, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap2, mockProtocol)
		if p1 == nil || p2 == nil {
			t.Fatal("get conn pool failed")
		}
		// cluster1 defaultPool, cluster2 clusterPool
		require.True(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		require.True(t, globalPoolExists(h2.Address))
		require.False(t, clusterPoolExists(h2.Address, snap2))
		clusterManagerInstance.ShutdownConnectionPool(mockProtocol, h.Address)
		require.False(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		require.False(t, globalPoolExists(h2.Address))
		require.False(t, clusterPoolExists(h2.Address, snap2))

		p1, _ = GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
		p2, _ = GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap2, mockProtocol)
		if p1 == nil || p2 == nil {
			t.Fatal("get conn pool failed")
		}
		// cluster1 defaultPool, cluster2 clusterPool
		require.True(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		require.True(t, globalPoolExists(h2.Address))
		require.False(t, clusterPoolExists(h2.Address, snap2))
		clusterManagerInstance.ShutdownConnectionPool("", h.Address)
		require.False(t, globalPoolExists(h.Address))
		require.False(t, clusterPoolExists(h.Address, snap1))
		require.False(t, globalPoolExists(h2.Address))
		require.False(t, clusterPoolExists(h2.Address, snap2))
	})
}

func _createClusterManagerWithConfig(config *v2.ClusterManagerConfig) types.ClusterManager {
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
	}, config)
}
