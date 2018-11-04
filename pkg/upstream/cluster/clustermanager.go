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
	"net"
	"reflect"
	"sync"

	"github.com/alipay/sofa-mosn/pkg/admin"
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/proxy"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var instanceMutex = sync.Mutex{}
var clusterMangerInstance *clusterManager

// ClusterManager
type clusterManager struct {
	sourceAddr             net.Addr
	primaryClusters        sync.Map // string: *primaryCluster
	protocolConnPool       sync.Map
	autoDiscovery          bool
	registryUseHealthCheck bool
}

type clusterSnapshot struct {
	prioritySet  types.PrioritySet
	clusterInfo  types.ClusterInfo
	loadbalancer types.LoadBalancer
}

func NewClusterManager(sourceAddr net.Addr, clusters []v2.Cluster,
	clusterMap map[string][]v2.Host, autoDiscovery bool, useHealthCheck bool) types.ClusterManager {
	instanceMutex.Lock()
	defer instanceMutex.Unlock()
	if clusterMangerInstance != nil {
		return clusterMangerInstance
	}

	clusterMangerInstance = &clusterManager{
		sourceAddr:       sourceAddr,
		primaryClusters:  sync.Map{},
		protocolConnPool: sync.Map{},
		autoDiscovery:    true, //todo delete
	}

	for k := range types.ConnPoolFactories {
		clusterMangerInstance.protocolConnPool.Store(k, &sync.Map{})
	}

	//init clusterMngInstance when run app
	initClusterMngAdapterInstance(clusterMangerInstance)

	//Add cluster to cm
	//Register upstream update type
	for _, cluster := range clusters {

		if !clusterMangerInstance.AddOrUpdatePrimaryCluster(cluster) {
			log.DefaultLogger.Errorf("NewClusterManager: AddOrUpdatePrimaryCluster failure, cluster name = %s", cluster.Name)
		}
	}

	// Add hosts to cluster
	// Note: currently, use priority = 0
	for clusterName, hosts := range clusterMap {
		clusterMangerInstance.UpdateClusterHosts(clusterName, 0, hosts)
	}

	return clusterMangerInstance
}

func (cs *clusterSnapshot) PrioritySet() types.PrioritySet {
	return cs.prioritySet
}

func (cs *clusterSnapshot) ClusterInfo() types.ClusterInfo {
	return cs.clusterInfo
}

func (cs *clusterSnapshot) LoadBalancer() types.LoadBalancer {
	return cs.loadbalancer
}

type primaryCluster struct {
	cluster     types.Cluster
	addedViaAPI bool
	configUsed  *v2.Cluster // used for update
}

// AddOrUpdatePrimaryCluster
// used to "add" cluster if cluster not exist
// or "update" cluster when new cluster config if cluster already exist
func (cm *clusterManager) AddOrUpdatePrimaryCluster(cluster v2.Cluster) bool {
	clusterName := cluster.Name

	ok := false
	if v, exist := cm.primaryClusters.Load(clusterName); exist {
		if !v.(*primaryCluster).addedViaAPI {
			return false
		}
		// update cluster
		ok = cm.updateCluster(cluster, v.(*primaryCluster), true)
	} else {
		// add new cluster
		ok = cm.loadCluster(cluster, true)
	}
	if ok {
		admin.SetClusterConfig(clusterName, cluster)
	}
	return ok
}

func (cm *clusterManager) ClusterExist(clusterName string) bool {
	if _, exist := cm.primaryClusters.Load(clusterName); exist {
		return true
	}

	return false
}

func (cm *clusterManager) updateCluster(clusterConf v2.Cluster, pcluster *primaryCluster, addedViaAPI bool) bool {
	if reflect.DeepEqual(clusterConf, pcluster.configUsed) {
		log.DefaultLogger.Debugf("update cluster but get duplicate configure")
		return true
	}

	if concretedCluster, ok := pcluster.cluster.(*simpleInMemCluster); ok {
		hosts := concretedCluster.hosts
		cluster := NewCluster(clusterConf, cm.sourceAddr, addedViaAPI)
		cluster.(*simpleInMemCluster).UpdateHosts(hosts)
		cm.primaryClusters.Store(clusterConf.Name, &primaryCluster{
			cluster:     cluster,
			addedViaAPI: addedViaAPI,
		})

		return true
	}

	return false
}

func (cm *clusterManager) loadCluster(clusterConfig v2.Cluster, addedViaAPI bool) bool {
	//clusterConfig.UseHealthCheck
	cluster := NewCluster(clusterConfig, cm.sourceAddr, addedViaAPI)

	if nil == cluster {
		return false
	}

	cluster.Initialize(func() {
		cluster.PrioritySet().AddMemberUpdateCb(func(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
		})
	})

	cm.primaryClusters.Store(clusterConfig.Name, &primaryCluster{
		cluster:     cluster,
		addedViaAPI: addedViaAPI,
		configUsed:  &clusterConfig,
	})

	return true
}

func (cm *clusterManager) getOrCreateClusterSnapshot(clusterName string) *clusterSnapshot {
	if v, ok := cm.primaryClusters.Load(clusterName); ok {
		pcc := v.(*primaryCluster).cluster

		clusterSnapshot := &clusterSnapshot{
			prioritySet:  pcc.PrioritySet(),
			clusterInfo:  pcc.Info(),
			loadbalancer: pcc.Info().LBInstance(),
		}

		return clusterSnapshot
	}

	return nil
}

func (cm *clusterManager) RemovePrimaryCluster(clusterName string) error {
	if v, exist := cm.primaryClusters.Load(clusterName); exist {
		if !v.(*primaryCluster).addedViaAPI {
			return fmt.Errorf("Remove Primary Cluster Failed, Cluster Name = %s not addedViaAPI", clusterName)
		}
		cm.primaryClusters.Delete(clusterName)
		log.DefaultLogger.Debugf("Remove Primary Cluster, Cluster Name = %s", clusterName)
		return nil
	}

	return fmt.Errorf("Remove Primary Cluster failure, cluster name = %s doesn't exist", clusterName)
}

func (cm *clusterManager) SetInitializedCb(cb func()) {}

func (cm *clusterManager) Clusters() map[string]types.Cluster {
	clusterInfoMap := make(map[string]types.Cluster)

	cm.primaryClusters.Range(func(key, value interface{}) bool {
		clusterInfoMap[key.(string)] = value.(*primaryCluster).cluster
		return true
	})

	return clusterInfoMap
}

func (cm *clusterManager) Get(context context.Context, cluster string) types.ClusterSnapshot {
	return cm.getOrCreateClusterSnapshot(cluster)
}

func (cm *clusterManager) UpdateClusterHosts(clusterName string, priority uint32, hostConfigs []v2.Host) error {
	if v, ok := cm.primaryClusters.Load(clusterName); ok {
		pcc := v.(*primaryCluster).cluster

		// todo: hack
		if concretedCluster, ok := pcc.(*simpleInMemCluster); ok {
			var hosts []types.Host

			for _, hc := range hostConfigs {
				hosts = append(hosts, NewHost(hc, pcc.Info()))
			}
			concretedCluster.UpdateHosts(hosts)
			admin.SetHosts(clusterName, hostConfigs)

			return nil
		}

		return fmt.Errorf("UpdateClusterHosts failed, cluster's hostset %s can't be update", clusterName)
	}

	return fmt.Errorf("UpdateClusterHosts failed, cluster %s not found", clusterName)
}

func (cm *clusterManager) RemoveClusterHost(clusterName string, hostAddress string) error {
	if hostAddress == "" {
		return fmt.Errorf("RemoveClusterHost failed, hostAddress is nil")
	}

	if v, ok := cm.primaryClusters.Load(clusterName); ok {
		pcc := v.(*primaryCluster).cluster

		found := false
		if concretedCluster, ok := pcc.(*simpleInMemCluster); ok {
			//ccHosts := concretedCluster.hosts
			for i := 0; i < len(concretedCluster.hosts); i++ {
				if hostAddress == concretedCluster.hosts[i].AddressString() {
					concretedCluster.hosts = append(concretedCluster.hosts[:i], concretedCluster.hosts[i+1:]...)
					found = true
					break
				}
			}
			if found == true {
				log.DefaultLogger.Debugf("RemoveClusterHost success, host address = %s", hostAddress)
				//	concretedCluster.UpdateHosts(ccHosts)
				return nil
			}
			return fmt.Errorf("RemoveClusterHost failed, host address = %s doesn't exist", hostAddress)

		}

		return fmt.Errorf("RemoveClusterHost failed, cluster name = %s is not valid", clusterName)
	}

	return fmt.Errorf("RemoveClusterHost failed, cluster name = %s doesn't exist", clusterName)
}

func (cm *clusterManager) TCPConnForCluster(lbCtx types.LoadBalancerContext, cluster string) types.CreateConnectionData {
	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster)

	if clusterSnapshot == nil {
		return types.CreateConnectionData{}
	}

	host := clusterSnapshot.loadbalancer.ChooseHost(lbCtx)

	if host != nil {
		return host.CreateConnection(nil)
	}

	return types.CreateConnectionData{}
}

func (cm *clusterManager) ConnPoolForCluster(balancerContext types.LoadBalancerContext, cluster string, protocol types.Protocol) types.ConnectionPool {

	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster)

	if clusterSnapshot == nil {
		log.DefaultLogger.Errorf(" %s ConnPool For Cluster is nil, cluster name = %s", protocol, cluster)
		return nil
	}

	host := clusterSnapshot.loadbalancer.ChooseHost(balancerContext)

	if host != nil {
		addr := host.AddressString()
		log.DefaultLogger.Debugf(" clusterSnapshot.loadbalancer.ChooseHost result is %s, cluster name = %s", addr, cluster)

		value, _ := cm.protocolConnPool.Load(protocol)

		connectionPool := value.(*sync.Map)
		if connPool, ok := connectionPool.Load(addr); ok {
			return connPool.(types.ConnectionPool)
		}
		if factory, ok := proxy.ConnNewPoolFactories[protocol]; ok {
			newPool := factory(host) //call NewBasicRoute

			connectionPool.Store(addr, newPool)

			return newPool
		}
	}

	log.DefaultLogger.Errorf("clusterSnapshot.loadbalancer.ChooseHost is nil, cluster name = %s", cluster)
	return nil
}

func (cm *clusterManager) Shutdown() error {
	return nil
}

func (cm *clusterManager) SourceAddress() net.Addr {
	return cm.sourceAddr
}

func (cm *clusterManager) VersionInfo() string {
	return ""
}

func (cm *clusterManager) LocalClusterName() string {
	return ""
}

// Destory the cluster manager instance
func (cm *clusterManager) Destory() {
	instanceMutex.Lock()
	defer instanceMutex.Unlock()
	if clusterMangerInstance != nil {
		clusterMangerInstance = nil
	}
}
