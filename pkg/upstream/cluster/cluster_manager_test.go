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

	"github.com/stretchr/testify/require"
	v2 "mosn.io/mosn/pkg/config/v2"
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

}
