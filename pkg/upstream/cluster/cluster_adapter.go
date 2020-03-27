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
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

// cluster adapter wrapper cluster manager
// the adapter can call clustermanager directly
// but at most time, we call the wrapper functions
type MngAdapter struct {
	types.ClusterManager
}

var adapterInstance = &MngAdapter{
	ClusterManager: clusterManagerInstance,
}

func GetClusterMngAdapterInstance() *MngAdapter {
	return adapterInstance
}

func (ca *MngAdapter) TriggerClusterAddOrUpdate(cluster v2.Cluster) error {
	return ca.AddOrUpdatePrimaryCluster(cluster)
}

func (ca *MngAdapter) TriggerClusterAndHostsAddOrUpdate(cluster v2.Cluster, hosts []v2.Host) error {
	if err := ca.AddOrUpdatePrimaryCluster(cluster); err != nil {
		return err
	}
	return ca.UpdateClusterHosts(cluster.Name, hosts)
}

func (ca *MngAdapter) TriggerClusterDel(clusterNames ...string) error {
	return ca.RemovePrimaryCluster(clusterNames...)
}

func (ca *MngAdapter) TriggerClusterHostUpdate(clusterName string, hosts []v2.Host) error {
	return ca.UpdateClusterHosts(clusterName, hosts)
}

func (ca *MngAdapter) TriggerHostDel(clusterName string, hosts []string) error {
	return ca.RemoveClusterHosts(clusterName, hosts)
}

func (ca *MngAdapter) TriggerHostAppend(clusterName string, hostAppend []v2.Host) error {
	return ca.AppendClusterHosts(clusterName, hostAppend)
}
