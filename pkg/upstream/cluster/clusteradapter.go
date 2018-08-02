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
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
)

// Adap is the instance of cluster Adapter
var Adap Adapter

type Adapter struct {
	clusterMng *clusterManager
}

// TriggerClusterAddedOrUpdate
// Added or Update Cluster, but not for cluster's hosts
func (ca *Adapter) TriggerClusterAddedOrUpdate(cluster v2.Cluster) error {
	clusterExist := ca.clusterMng.ClusterExist(cluster.Name)

	// add cluster
	if !clusterExist {
		log.DefaultLogger.Debugf("Add PrimaryCluster: %s", cluster.Name)

		// for dynamically added cluster, use cluster manager's health check config
		if ca.clusterMng.registryUseHealthCheck {
			cluster.HealthCheck = sofarpc.DefaultSofaRPCHealthCheckConf
		}

		if !ca.clusterMng.AddOrUpdatePrimaryCluster(cluster) {
			return fmt.Errorf("TriggerClusterAdded: AddOrUpdatePrimaryCluster failure, cluster name = %s", cluster.Name)
		}
	} else {
		log.DefaultLogger.Debugf("PrimaryCluster Already Exist: %s", cluster.Name)
	}

	return nil
}

// TriggerClusterAddedOrUpdate
// Added or Update Cluster and Cluster's hosts
func (ca *Adapter) TriggerClusterAndHostsAddedOrUpdate(cluster v2.Cluster, hosts []v2.Host) error {
	if err := ca.TriggerClusterAddedOrUpdate(cluster); err != nil {
		return err
	}
	return ca.clusterMng.UpdateClusterHosts(cluster.Name, 0, hosts)
}

// TriggerClusterHostUpdate
// Added or Update Cluster's hosts, return err if cluster not exist
func (ca *Adapter) TriggerClusterHostUpdate(clusterName string, hosts []v2.Host) error {
	return ca.clusterMng.UpdateClusterHosts(clusterName, 0, hosts)
}

// TriggerClusterDel
// used to delete cluster
func (ca *Adapter) TriggerClusterDel(clusterName string) {
	log.DefaultLogger.Debugf("Delete Cluster %s", clusterName)
	ca.clusterMng.RemovePrimaryCluster(clusterName)
}
