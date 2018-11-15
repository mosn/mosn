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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var clusterMngAdapterInstance *MngAdapter

func initClusterMngAdapterInstance(clusterMng *clusterManager) {
	clusterMngAdapterInstance = &MngAdapter{
		clusterMng: clusterMng,
	}
}

// GetClusterMngAdapterInstance used to get clusterMngAdapterInstance
func GetClusterMngAdapterInstance() *MngAdapter {
	return clusterMngAdapterInstance
}

// MngAdapter is the wrapper or adapter for external caller
type MngAdapter struct {
	clusterMng *clusterManager
}

// TriggerClusterAddOrUpdate used to Added or Update Cluster
func (ca *MngAdapter) TriggerClusterAddOrUpdate(cluster v2.Cluster) error {
	if ca.clusterMng == nil {
		return fmt.Errorf("TriggerClusterAddOrUpdate Error: cluster manager is nil")
	}

	if !ca.clusterMng.AddOrUpdatePrimaryCluster(cluster) {
		return fmt.Errorf("TriggerClusterAddOrUpdate failure, cluster name = %s", cluster.Name)
	}

	return nil
}

// TriggerClusterAndHostsAddOrUpdate used to Added or Update Cluster and Cluster's hosts
func (ca *MngAdapter) TriggerClusterAndHostsAddOrUpdate(cluster v2.Cluster, hosts []v2.Host) error {
	if err := ca.TriggerClusterAddOrUpdate(cluster); err != nil {
		return err
	}

	return ca.clusterMng.UpdateClusterHosts(cluster.Name, 0, hosts)
}

// TriggerClusterDel :used to delete c uster by clusterName
func (ca *MngAdapter) TriggerClusterDel(clusterName string) error {
	if ca.clusterMng == nil {
		return fmt.Errorf("TriggerClusterAddOrUpdate Error: cluster manager is nil")
	}

	return ca.clusterMng.RemovePrimaryCluster(clusterName)
}

// TriggerClusterHostUpdate used to Added or Update Cluster's hosts, return err if cluster not exist
func (ca *MngAdapter) TriggerClusterHostUpdate(clusterName string, hosts []v2.Host) error {
	if ca.clusterMng == nil {
		return fmt.Errorf("TriggerClusterAddOrUpdate Error: cluster manager is nil")
	}

	return ca.clusterMng.UpdateClusterHosts(clusterName, 0, hosts)
}

// TriggerHostDel used to delete
func (ca *MngAdapter) TriggerHostDel(clusterName string, hostAddress string) error {
	return ca.clusterMng.RemoveClusterHost(clusterName, hostAddress)
}

// GetCluster used to get cluster by name
func (ca *MngAdapter) GetClusterSnapshot(context context.Context, clusterName string) types.ClusterSnapshot {
	return ca.clusterMng.GetClusterSnapshot(context, clusterName)
}

// PutClusterSnapshot used to put cluster snapshot, release rcu
func (ca *MngAdapter) PutClusterSnapshot(snapshot types.ClusterSnapshot) {
	ca.clusterMng.PutClusterSnapshot(snapshot)
	return
}
