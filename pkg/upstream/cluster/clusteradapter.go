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
	"errors"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
)

var Adap Adapter

type Adapter struct {
	clusterMng *clusterManager
}

// Called by registry module to update cluster's host info
func (ca *Adapter) TriggerClusterUpdate(clusterName string, hosts []v2.Host) error {
	clusterExist := ca.clusterMng.ClusterExist(clusterName)

	if !clusterExist {
		if ca.clusterMng.autoDiscovery {
			cluster := v2.Cluster{
				Name:        clusterName,
				ClusterType: v2.DYNAMIC_CLUSTER,
				LbType:      v2.LB_RANDOM,
			}

			// for dynamically added cluster, use cluster manager's health check config
			if ca.clusterMng.registryUseHealthCheck {
				// todo support more default health check @boqin
				cluster.HealthCheck = sofarpc.DefaultSofaRpcHealthCheckConf
			}

			ca.clusterMng.AddOrUpdatePrimaryCluster(cluster)
		} else {
			msg := "cluster doesn't support auto discovery "
			log.DefaultLogger.Errorf(msg)
			return errors.New(msg)
		}
	}

	log.DefaultLogger.Debugf("triggering cluster update, cluster name = %s hosts = %+v", clusterName, hosts)
	ca.clusterMng.UpdateClusterHosts(clusterName, 0, hosts)

	return nil
}

// Called when mesh receive subscribe info
func (ca *Adapter) TriggerClusterAdded(cluster v2.Cluster) {
	clusterExist := ca.clusterMng.ClusterExist(cluster.Name)

	if !clusterExist {
		log.DefaultLogger.Debugf("Add PrimaryCluster: %s", cluster.Name)

		// for dynamically added cluster, use cluster manager's health check config
		if ca.clusterMng.registryUseHealthCheck {
			cluster.HealthCheck = sofarpc.DefaultSofaRpcHealthCheckConf
		}

		ca.clusterMng.AddOrUpdatePrimaryCluster(cluster)
	} else {
		log.DefaultLogger.Debugf("Added PrimaryCluster: %s Already Exist", cluster.Name)
	}
}

// Called when mesh receive unsubscribe info
func (ca *Adapter) TriggerClusterDel(clusterName string) {
	log.DefaultLogger.Debugf("Delete Cluster %s", clusterName)
	ca.clusterMng.RemovePrimaryCluster(clusterName)
}
