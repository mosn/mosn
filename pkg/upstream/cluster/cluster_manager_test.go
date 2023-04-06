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
	_createClusterManager()
	config := v2.Cluster{
		Name:              "test1",
		ClusterPoolEnable: true,
	}
	GetClusterMngAdapterInstance().AddOrUpdatePrimaryCluster(config)
	snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	mockLbCtx := newMockLbContext(nil)
	p, _ := GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
	if p == nil {
		t.Fatal("get conn pool failed")
	}
}

func TestClusterManager_ShutdownConnectionPool(t *testing.T) {
	_createClusterManager()
	h := v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1:10000",
		},
	}
	GetClusterMngAdapterInstance().UpdateClusterHosts("test1", []v2.Host{h})
	snap1 := GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	mockLbCtx := newMockLbContext(nil)
	GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
	m, _ := GetClusterMngAdapterInstance().ClusterManager.(*clusterManagerSingleton).protocolConnPool.Load(mockProtocol)
	_, ok := m.(*sync.Map).Load(h.Address)
	require.True(t, ok)
	clusterManagerInstance.ShutdownConnectionPool(mockProtocol, snap1.HostSet().Get(0).AddressString())
	_, ok = m.(*sync.Map).Load(h.Address)
	require.False(t, ok)

	GetClusterMngAdapterInstance().Destroy()
	_createClusterManager()
	GetClusterMngAdapterInstance().AddOrUpdateClusterAndHost(v2.Cluster{
		Name:              "test1",
		ClusterPoolEnable: true,
	}, []v2.Host{h})
	snap1 = GetClusterMngAdapterInstance().GetClusterSnapshot(context.Background(), "test1")
	GetClusterMngAdapterInstance().ConnPoolForCluster(mockLbCtx, snap1, mockProtocol)
	m, _ = GetClusterMngAdapterInstance().ClusterManager.(*clusterManagerSingleton).protocolConnPool.Load(mockProtocol)
	_, ok = m.(*sync.Map).Load(h.Address)
	require.False(t, ok)
	clusterPoolMap, ok := m.(*sync.Map).Load(snap1.ClusterInfo().Name())
	require.True(t, ok)
	_, ok = clusterPoolMap.(*sync.Map).Load(h.Address)
	require.True(t, ok)

	clusterManagerInstance.ShutdownConnectionPool(mockProtocol, snap1.HostSet().Get(0).AddressString())
	_, ok = m.(*sync.Map).Load(h.Address)
	require.False(t, ok)
	clusterPoolMap, _ = m.(*sync.Map).Load(snap1.ClusterInfo().Name())
	_, ok = clusterPoolMap.(*sync.Map).Load(h.Address)
	require.False(t, ok)
}
