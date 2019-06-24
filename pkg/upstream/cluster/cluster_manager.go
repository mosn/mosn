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
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"sofastack.io/sofa-mosn/pkg/admin/store"
	"sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/network"
	"sofastack.io/sofa-mosn/pkg/rcu"
	"sofastack.io/sofa-mosn/pkg/types"
)

// a cluster wrapper for cluster manager
type primaryCluster struct {
	cluster    types.Cluster
	updateLock sync.Mutex
	configLock *rcu.Value //FIXME: rcu value store typs.Cluster
}

func newPrimaryCluster(cluster types.Cluster) *primaryCluster {
	cfg := true
	return &primaryCluster{
		cluster:    cluster,
		configLock: rcu.NewValue(&cfg),
	}
}

var errNilCluster = errors.New("cannot update nil cluster")

func (pc *primaryCluster) UpdateCluster(cluster types.Cluster) error {
	if cluster == nil || reflect.ValueOf(cluster).IsNil() {
		return errNilCluster
	}
	pc.updateLock.Lock()
	defer pc.updateLock.Unlock()
	pc.cluster = cluster
	cfg := true
	if err := pc.configLock.Update(&cfg, 0); err == rcu.Block {
		return err
	}
	log.DefaultLogger.Infof("[cluster] [primaryCluster] [UpdateHosts] cluster %s update", pc.cluster.Info().Name())
	return nil
}

func (pc *primaryCluster) refreshHostsConfig() {
	// set Hosts Config
	hosts := pc.cluster.HostSet().Hosts()
	hostsConfig := make([]v2.Host, 0, len(hosts))
	for _, h := range hosts {
		hostsConfig = append(hostsConfig, h.Config())
	}
	store.SetHosts(pc.cluster.Info().Name(), hostsConfig)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[cluster] [primaryCluster] [UpdateHosts] cluster %s update hosts: %d", pc.cluster.Info().Name(), len(hosts))
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Infof("[cluster] [primaryCluster] [UpdateHosts] cluster %s update hosts: %v", pc.cluster.Info().Name(), hosts)
	}
}

func (pc *primaryCluster) UpdateHosts(hosts []types.Host) error {
	pc.updateLock.Lock()
	defer pc.updateLock.Unlock()
	pc.cluster.UpdateHosts(hosts)
	// rcu lock
	cfg := true
	if err := pc.configLock.Update(&cfg, 0); err == rcu.Block {
		return err
	}
	pc.refreshHostsConfig()
	return nil

}

func (pc *primaryCluster) AppendHost(hosts []types.Host) error {
	pc.updateLock.Lock()
	defer pc.updateLock.Unlock()
	// Append Hosts
	allHosts := pc.cluster.HostSet().Hosts()
	hosts = append(hosts, allHosts...)
	pc.cluster.UpdateHosts(hosts)
	cfg := true
	if err := pc.configLock.Update(&cfg, 0); err == rcu.Block {
		return err
	}
	pc.refreshHostsConfig()
	return nil
}

func (pc *primaryCluster) RemoveHosts(addrs []string) error {
	pc.updateLock.Lock()
	defer pc.updateLock.Unlock()
	// Remove Hosts
	pc.cluster.RemoveHosts(addrs)
	cfg := true
	if err := pc.configLock.Update(&cfg, 0); err == rcu.Block {
		return err
	}
	pc.refreshHostsConfig()
	return nil
}

// types.ClusterManager
type clusterManager struct {
	primaryClusterMap sync.Map // string: *primaryCluster
	protocolConnPool  sync.Map
	mux               sync.Mutex
}

type clusterManagerSingleton struct {
	instanceMutex sync.Mutex
	*clusterManager
}

func (singleton *clusterManagerSingleton) Destroy() {
	clusterMangerInstance.instanceMutex.Lock()
	defer clusterMangerInstance.instanceMutex.Unlock()
	clusterMangerInstance.clusterManager = nil
}

var clusterMangerInstance = &clusterManagerSingleton{}

func NewClusterManagerSingleton(clusters []v2.Cluster, clusterMap map[string][]v2.Host) types.ClusterManager {
	clusterMangerInstance.instanceMutex.Lock()
	defer clusterMangerInstance.instanceMutex.Unlock()
	if clusterMangerInstance.clusterManager != nil {
		return clusterMangerInstance
	}
	clusterMangerInstance.clusterManager = &clusterManager{}
	for k := range types.ConnPoolFactories {
		clusterMangerInstance.protocolConnPool.Store(k, &sync.Map{})
	}

	//Add cluster to cm
	for _, cluster := range clusters {
		if err := clusterMangerInstance.AddOrUpdatePrimaryCluster(cluster); err != nil {
			log.DefaultLogger.Errorf("[upstream] [cluster manager] NewClusterManager: AddOrUpdatePrimaryCluster failure, cluster name = %s, error: %v", cluster.Name, err)
		}
	}
	// Add cluster host
	for clusterName, hosts := range clusterMap {
		if err := clusterMangerInstance.UpdateClusterHosts(clusterName, hosts); err != nil {
			log.DefaultLogger.Errorf("[upstream] [cluster manager] NewClusterManager: UpdateClusterHosts failure, cluster name = %s, error: %v", clusterName, err)
		}
	}
	return clusterMangerInstance

}

// AddOrUpdatePrimaryCluster will always create a new cluster without the hosts config
// if the same name cluster is already exists, we will keep the exists hosts, and use rcu to update it.
func (cm *clusterManager) AddOrUpdatePrimaryCluster(cluster v2.Cluster) error {
	// new cluster
	newCluster := NewCluster(cluster)
	if newCluster == nil || reflect.ValueOf(newCluster).IsNil() {
		return errNilCluster
	}
	// check update or new
	clusterName := cluster.Name
	pci, loaded := cm.primaryClusterMap.LoadOrStore(clusterName, newPrimaryCluster(newCluster))
	if loaded { // update
		pc := pci.(*primaryCluster)
		//FIXME: cluster info in hosts should be updated too
		hosts := pc.cluster.HostSet().Hosts()
		newCluster.UpdateHosts(hosts)
		if err := pc.UpdateCluster(newCluster); err != nil {
			return err
		}
	}
	store.SetClusterConfig(clusterName, cluster)
	log.DefaultLogger.Infof("[cluster] [cluster manager] [AddOrUpdatePrimaryCluster] cluster %s updated", clusterName)
	return nil
}

// AddClusterHealthCheckCallbacks adds a health check callback function into cluster
func (cm *clusterManager) AddClusterHealthCheckCallbacks(name string, cb types.HealthCheckCb) error {
	pci, ok := cm.primaryClusterMap.Load(name)
	if ok {
		pc := pci.(*primaryCluster)
		pc.cluster.AddHealthCheckCallbacks(cb)
		return nil
	}
	return fmt.Errorf("cluster %s is not exists", name)
}

func (cm *clusterManager) ClusterExist(clusterName string) bool {
	_, ok := cm.primaryClusterMap.Load(clusterName)
	return ok
}

// RemovePrimaryCluster removes clusters from cluster manager
// If the cluster is more than one, all of them should be exists, or no one will be deleted
func (cm *clusterManager) RemovePrimaryCluster(clusterNames ...string) error {
	// check all clutsers in cluster manager
	for _, clusterName := range clusterNames {
		_, ok := cm.primaryClusterMap.Load(clusterName)
		if !ok {
			return fmt.Errorf("remove cluster failed, cluster %s is not exists", clusterName)
		}
	}
	// delete all of them
	for _, clusterName := range clusterNames {
		cm.primaryClusterMap.Delete(clusterName)
		store.RemoveClusterConfig(clusterName)
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[upstream] [cluster manager] Remove Primary Cluster, Cluster Name = %s", clusterName)
		}
	}
	return nil
}

// UpdateClusterHosts update all hosts in the cluster
func (cm *clusterManager) UpdateClusterHosts(clusterName string, hostConfigs []v2.Host) error {
	pci, ok := cm.primaryClusterMap.Load(clusterName)
	if !ok {
		return fmt.Errorf("cluster %s is not exists", clusterName)
	}
	pc := pci.(*primaryCluster)
	var hosts []types.Host
	for _, hc := range hostConfigs {
		hosts = append(hosts, NewSimpleHost(hc, pc.cluster.Info()))
	}
	if err := pc.UpdateHosts(hosts); err != nil {
		log.DefaultLogger.Errorf("[upstream] [cluster manager] update cluster %s hosts failed, error: %v", clusterName, err)
		return err
	}
	return nil
}

// AppendClusterHosts adds a new host into cluster
func (cm *clusterManager) AppendClusterHosts(clusterName string, hostConfigs []v2.Host) error {
	pci, ok := cm.primaryClusterMap.Load(clusterName)
	if !ok {
		return fmt.Errorf("cluster %s is not exists", clusterName)
	}
	pc := pci.(*primaryCluster)
	var hosts []types.Host
	for _, hc := range hostConfigs {
		hosts = append(hosts, NewSimpleHost(hc, pc.cluster.Info()))
	}
	if err := pc.AppendHost(hosts); err != nil {
		log.DefaultLogger.Errorf("[upstream] [cluster manager] append hosts into cluster %s failed, error:%v", clusterName, err)
		return err
	}
	return nil
}

func (cm *clusterManager) RemoveClusterHosts(clusterName string, hosts []string) error {
	pci, ok := cm.primaryClusterMap.Load(clusterName)
	if !ok {
		return fmt.Errorf("cluster %s is not exists", clusterName)
	}
	pc := pci.(*primaryCluster)
	if err := pc.RemoveHosts(hosts); err != nil {
		log.DefaultLogger.Errorf("[upstream] [cluster manager] remove host %v from cluster %s failed, error: %v", hosts, clusterName, err)
		return err
	}
	return nil
}

func (cm *clusterManager) GetClusterSnapshot(context context.Context, clusterName string) types.ClusterSnapshot {
	pci, ok := cm.primaryClusterMap.Load(clusterName)
	if !ok {
		return nil
	}
	pc := pci.(*primaryCluster)
	return &clusterSnapshot{
		cluster: pc.cluster,
		value:   pc.configLock,
		config:  pc.configLock.Load(),
	}
}

func (cm *clusterManager) PutClusterSnapshot(snapshot types.ClusterSnapshot) {
	if snapshot == nil || reflect.ValueOf(snapshot).IsNil() {
		return
	}
	s, ok := snapshot.(*clusterSnapshot)
	if !ok {
		log.DefaultLogger.Errorf("[upstream] [cluster manager] snapshot is not clusterSnapshot, clustername=%s", snapshot.ClusterInfo().Name())
		return
	}
	s.value.Put(s.config)
}

func (cm *clusterManager) TCPConnForCluster(lbCtx types.LoadBalancerContext, snapshot types.ClusterSnapshot) types.CreateConnectionData {
	if snapshot == nil || reflect.ValueOf(snapshot).IsNil() {
		return types.CreateConnectionData{}
	}
	host := snapshot.LoadBalancer().ChooseHost(lbCtx)
	if host == nil {
		return types.CreateConnectionData{}
	}
	return host.CreateConnection(context.Background())
}

func (cm *clusterManager) ConnPoolForCluster(balancerContext types.LoadBalancerContext, snapshot types.ClusterSnapshot, protocol types.Protocol) types.ConnectionPool {
	if snapshot == nil || reflect.ValueOf(snapshot).IsNil() {
		log.DefaultLogger.Errorf("[upstream] [cluster manager]  %s ConnPool For Cluster is nil", protocol)
		return nil
	}
	pool, err := cm.getActiveConnectionPool(balancerContext, snapshot, protocol)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [cluster manager] ConnPoolForCluster Failed; %v", err)
	}
	return pool
}

const cycleTimes = 5

var (
	errNilHostChoose   = errors.New("cluster snapshot choose host is nil")
	errUnknownProtocol = errors.New("protocol pool can not found protocol")
	errNoHealthyHost   = errors.New("no health hosts")
)

func (cm *clusterManager) getActiveConnectionPool(balancerContext types.LoadBalancerContext, clusterSnapshot types.ClusterSnapshot, protocol types.Protocol) (types.ConnectionPool, error) {
	var pool types.ConnectionPool
	var pools [cycleTimes]types.ConnectionPool

	for i := 0; i < cycleTimes; i++ {
		host := clusterSnapshot.LoadBalancer().ChooseHost(balancerContext)
		if host == nil {
			return nil, errNilHostChoose
		}
		addr := host.AddressString()
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[upstream] [cluster manager] clusterSnapshot.loadbalancer.ChooseHost result is %s, cluster name = %s", addr, clusterSnapshot.ClusterInfo().Name())
		}
		value, ok := cm.protocolConnPool.Load(protocol)
		if !ok {
			return nil, errUnknownProtocol
		}
		connectionPool := value.(*sync.Map)
		connPool, ok := connectionPool.Load(addr)
		if ok {
			pool = connPool.(types.ConnectionPool)
			if pool.CheckAndInit(balancerContext.DownstreamContext()) {
				return pool, nil
			}
			pools[i] = pool
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[upstream] [cluster manager] cluster host %s is not active", addr)
			}
		} else {
			// connectionPool Load and Store should have concurrency control
			// we do not use LoadOrStore, so we can returns directly when protocol is not registered
			err := func() error {
				cm.mux.Lock()
				defer cm.mux.Unlock()
				if _, ok := connectionPool.Load(addr); ok {
					// if addr is already stored
					return nil
				}
				factory, ok := network.ConnNewPoolFactories[protocol]
				if !ok {
					return fmt.Errorf("protocol %v is not registered is pool factory", protocol)
				}
				newPool := factory(host)
				connectionPool.Store(addr, newPool)
				newPool.CheckAndInit(balancerContext.DownstreamContext())
				pools[i] = newPool
				return nil
			}()
			if err != nil {
				return nil, err
			}
		}
	}

	// perhaps the first request, wait for tcp handshaking. total wait time: 1ms + 10ms + 100ms + 1000ms
	waitTime := time.Millisecond
	for t := 0; t < 4; t++ {
		time.Sleep(waitTime)
		for i := 0; i < cycleTimes; i++ {
			if pools[i] == nil {
				continue
			}
			if pools[i].CheckAndInit(balancerContext.DownstreamContext()) {
				return pools[i], nil
			}
		}
		waitTime *= 10
	}
	return nil, errNoHealthyHost
}

type clusterSnapshot struct {
	cluster types.Cluster
	// rcu for snapshot
	value  *rcu.Value
	config interface{}
}

func (snapshot *clusterSnapshot) HostSet() types.HostSet {
	return snapshot.cluster.HostSet()
}

func (snapshot *clusterSnapshot) ClusterInfo() types.ClusterInfo {
	return snapshot.cluster.Info()
}

func (snapshot *clusterSnapshot) LoadBalancer() types.LoadBalancer {
	return snapshot.cluster.LBInstance()
}

func (snapshot *clusterSnapshot) IsExistsHosts(metadata types.MetadataMatchCriteria) bool {
	if sublb, ok := snapshot.cluster.LBInstance().(*subsetLoadBalancer); ok {
		if metadata != nil {
			matchCriteria := metadata.MetadataMatchCriteria()
			entry := sublb.findSubset(matchCriteria)
			empty := (entry == nil || !entry.Active())
			return !empty
		}
	}
	hosts := snapshot.cluster.HostSet().Hosts()
	return len(hosts) > 0

}
