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
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

var errNilCluster = errors.New("cannot update nil cluster")

// refreshHostsConfig refresh the stored config for admin api
func refreshHostsConfig(c types.Cluster) {
	// use new cluster snapshot to get new cluster config
	name := c.Snapshot().ClusterInfo().Name()
	hostSet := c.Snapshot().HostSet()
	hostsConfig := make([]v2.Host, 0, hostSet.Size())
	hostSet.Range(func(h types.Host) bool {
		hostsConfig = append(hostsConfig, h.Config())
		return true
	})
	configmanager.SetHosts(name, hostsConfig)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[cluster] [primaryCluster] [UpdateHosts] cluster %s update hosts: %d", name, hostSet.Size())
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[cluster] [primaryCluster] [UpdateHosts] cluster %s update hosts: %v", name, hostSet)
	}
}

const globalTLSMetrics = "global"

// types.ClusterManager
type clusterManager struct {
	clustersMap      sync.Map
	protocolConnPool *connPool
	tlsMetrics       *mtls.TLSStats
	tlsMng           atomic.Value // store types.TLSClientContextManager
	mux              sync.Mutex
}

type connPool struct {
	clusterPoolEnable bool     // TODO support modify clusterPoolEnable at runtime
	globalPool        sync.Map // proto: {addr: pool}
	clusterPool       sync.Map // proto: {cluster: {addr: pool}}
}

func newConnPool(clusterPoolEnable bool) *connPool {
	return &connPool{
		clusterPoolEnable: clusterPoolEnable,
	}
}

func (p *connPool) store(protocolName api.ProtocolName) {
	p.globalPool.Store(protocolName, &sync.Map{})
	p.clusterPool.Store(protocolName, &sync.Map{})
}

func (p *connPool) load(proto types.ProtocolName, snapshot types.ClusterSnapshot) (*sync.Map, bool) {
	var connectionPool *sync.Map
	if p.clusterPoolEnable || snapshot.ClusterInfo().IsClusterPoolEnable() {
		if poolMap, ok := p.clusterPool.Load(proto); ok {
			value, _ := poolMap.(*sync.Map).LoadOrStore(snapshot.ClusterInfo().Name(), &sync.Map{})
			connectionPool = value.(*sync.Map)
			return connectionPool, ok
		}
	} else {
		if poolMap, ok := p.globalPool.Load(proto); ok {
			connectionPool = poolMap.(*sync.Map)
			return connectionPool, ok
		}
	}
	return nil, false
}

func (p *connPool) shutdown(proto types.ProtocolName, addr string) {
	shutdownPool := func(value interface{}) {
		connectionPool := value.(*sync.Map)
		if cp, ok := connectionPool.Load(addr); ok {
			pool := cp.(types.ConnectionPool)
			connectionPool.Delete(addr)
			pool.Shutdown()
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[upstream] [cluster manager] protocol %s address %s connections shutdown", proto, addr)
			}
		}
	}
	if proto == "" {
		p.clusterPool.Range(func(_, clusterProtoPool interface{}) bool {
			clusterProtoPool.(*sync.Map).Range(func(_, connPool interface{}) bool {
				shutdownPool(connPool)
				return true
			})
			return true
		})
		p.globalPool.Range(func(_, connPool interface{}) bool {
			shutdownPool(connPool)
			return true
		})
	} else {
		clusterProtoPool, clusterExists := p.clusterPool.Load(proto)
		globalProtoPool, globalExists := p.globalPool.Load(proto)
		if !clusterExists || !globalExists {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[upstream] [cluster manager] unknown protocol when shutdown, protocol:%s, address: %s", proto, addr)
			}
			return
		}
		clusterProtoPool.(*sync.Map).Range(func(_, connPool interface{}) bool {
			shutdownPool(connPool)
			return true
		})
		shutdownPool(globalProtoPool)
	}
}

type clusterManagerSingleton struct {
	instanceMutex sync.Mutex
	*clusterManager
}

func (singleton *clusterManagerSingleton) Destroy() {
	clusterManagerInstance.instanceMutex.Lock()
	defer clusterManagerInstance.instanceMutex.Unlock()
	clusterManagerInstance.clusterManager = nil
}

var clusterManagerInstance = &clusterManagerSingleton{}

func NewClusterManagerSingleton(clusters []v2.Cluster, clusterMap map[string][]v2.Host, config *v2.ClusterManagerConfig) types.ClusterManager {
	clusterManagerInstance.instanceMutex.Lock()
	defer clusterManagerInstance.instanceMutex.Unlock()
	if clusterManagerInstance.clusterManager != nil {
		return clusterManagerInstance
	}

	clusterManagerInstance.clusterManager = &clusterManager{
		tlsMetrics: mtls.NewStats(globalTLSMetrics),
	}
	if config == nil {
		config = &v2.ClusterManagerConfig{}
	}
	// set global tls
	clusterManagerInstance.clusterManager.UpdateTLSManager(&config.TLSContext)
	// add conn pool
	clusterManagerInstance.protocolConnPool = newConnPool(config.ClusterPoolEnable)
	protocol.RangeAllRegisteredProtocol(func(k api.ProtocolName) {
		clusterManagerInstance.protocolConnPool.store(k)
	})

	//Add cluster to cm
	for _, cluster := range clusters {
		if err := clusterManagerInstance.AddOrUpdatePrimaryCluster(cluster); err != nil {
			log.DefaultLogger.Alertf("cluster.config", "[upstream] [cluster manager] NewClusterManager: AddOrUpdatePrimaryCluster failure, cluster name = %s, error: %v", cluster.Name, err)
		}
	}
	// Add cluster host
	for clusterName, hosts := range clusterMap {
		if err := clusterManagerInstance.UpdateClusterHosts(clusterName, hosts); err != nil {
			log.DefaultLogger.Alertf("cluster.config", "[upstream] [cluster manager] NewClusterManager: UpdateClusterHosts failure, cluster name = %s, error: %v", clusterName, err)
		}
	}
	return clusterManagerInstance
}

// CDS Handler
// old cluster maybe nil when cluster is newly created.
func UpdateClusterResourceManagerHandler(oc, nc types.Cluster) {
	if oc == nil {
		return
	}
	newSnap := nc.Snapshot()
	oldSnap := oc.Snapshot()

	if newSnap.ClusterInfo().ClusterType() != oldSnap.ClusterInfo().ClusterType() {
		return
	}

	newResourceManager := newSnap.ClusterInfo().ResourceManager()
	oldResourceManager := oldSnap.ClusterInfo().ResourceManager()

	// sync oldResourceManager to new cluster ResourceManager
	if ci, ok := newSnap.ClusterInfo().(*clusterInfo); ok {
		ci.resourceManager = oldResourceManager
	}
	// sync newResourceManager config to oldResourceManager
	updateResourceValue(oldResourceManager, newResourceManager)
}

func CleanOldClusterHandler(oc, _ types.Cluster) {
	if oc == nil {
		return
	}
	oc.StopHealthChecking()
}

func InheritClusterHostsHandler(oc, nc types.Cluster) {
	if oc == nil {
		return
	}

	newInfo := nc.Snapshot().ClusterInfo()
	oc.Snapshot().HostSet().Range(func(host types.Host) bool {
		host.SetClusterInfo(newInfo) // update host cluster info
		return true
	})
	nc.UpdateHosts(oc.Snapshot().HostSet())
}

func transferHostSetStates(os, ns types.HostSet) {
	if ns.Size() == 0 {
		return
	}

	oldHosts := make(map[string]types.Host, os.Size())

	os.Range(func(host types.Host) bool {
		oldHosts[host.AddressString()] = host
		return true
	})

	now := time.Now()
	ns.Range(func(host types.Host) bool {
		if h, ok := oldHosts[host.AddressString()]; ok {
			host.SetLastHealthCheckPassTime(h.LastHealthCheckPassTime())
		} else {
			host.SetLastHealthCheckPassTime(now)
		}
		return true
	})
}

func TransferClusterHostStatesHandler(oc, nc types.Cluster) {
	if oc == nil {
		return
	}

	transferHostSetStates(oc.Snapshot().HostSet(), nc.Snapshot().HostSet())
}

// AddOrUpdatePrimaryCluster will always create a new cluster without the hosts config
// if the same name cluster is already exists, we will keep the exists hosts.
func (cm *clusterManager) AddOrUpdatePrimaryCluster(cluster v2.Cluster) error {
	return cm.UpdateCluster(cluster, func(oc, nc types.Cluster) {
		UpdateClusterResourceManagerHandler(oc, nc)
		CleanOldClusterHandler(oc, nc)
		InheritClusterHostsHandler(oc, nc)
	})
}

// AddOrUpdateClusterAndHost will create a new cluster and use hosts config replace the original hosts if cluster is exists
func (cm *clusterManager) AddOrUpdateClusterAndHost(cluster v2.Cluster, hostConfigs []v2.Host) error {
	return cm.UpdateCluster(cluster, func(oc, nc types.Cluster) {
		UpdateClusterResourceManagerHandler(oc, nc)
		CleanOldClusterHandler(oc, nc)
		NewSimpleHostHandler(nc, hostConfigs)
		if cluster.SlowStart.Mode != "" {
			TransferClusterHostStatesHandler(oc, nc)
		}
	})
}

func (cm *clusterManager) UpdateCluster(cluster v2.Cluster, clusterHandler types.ClusterUpdateHandler) error {
	// new cluster
	newCluster := NewCluster(cluster)
	if newCluster == nil || reflect.ValueOf(newCluster).IsNil() {
		log.DefaultLogger.Errorf("[cluster] [cluster manager] [AddOrUpdatePrimaryCluster] update cluster %s failed", cluster.Name)
		return errNilCluster
	}
	// check update or new
	clusterName := cluster.Name
	// set config
	configmanager.SetClusterConfig(cluster)
	// add or update
	ci, exists := cm.clustersMap.Load(clusterName)
	var oldCluster types.Cluster
	if exists {
		oldCluster = ci.(types.Cluster)
	}
	if clusterHandler != nil {
		clusterHandler(oldCluster, newCluster)
	}
	cm.clustersMap.Store(clusterName, newCluster)
	refreshHostsConfig(newCluster)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[cluster] [cluster manager] [AddOrUpdatePrimaryCluster] cluster %s updated", clusterName)
	}
	return nil

}

// AddClusterHealthCheckCallbacks adds a health check callback function into cluster
func (cm *clusterManager) AddClusterHealthCheckCallbacks(name string, cb types.HealthCheckCb) error {
	ci, ok := cm.clustersMap.Load(name)
	if ok {
		c := ci.(types.Cluster)
		c.AddHealthCheckCallbacks(cb)
		return nil
	}
	return fmt.Errorf("cluster %s is not exists", name)
}

func (cm *clusterManager) ClusterExist(clusterName string) bool {
	_, ok := cm.clustersMap.Load(clusterName)
	return ok
}

// RemovePrimaryCluster removes clusters from cluster manager
// If the cluster is more than one, all of them should exist, or no one will be deleted
func (cm *clusterManager) RemovePrimaryCluster(clusterNames ...string) error {
	// check all clusters in cluster manager
	for _, clusterName := range clusterNames {
		if _, ok := cm.clustersMap.Load(clusterName); !ok {
			log.DefaultLogger.Errorf("[upstream] [cluster manager] Remove Primary Cluster,  cluster %s is not exists", clusterName)
			return fmt.Errorf("remove cluster failed, cluster %s is not exists", clusterName)
		}
	}
	// delete all of them
	for _, clusterName := range clusterNames {
		v, ok := cm.clustersMap.Load(clusterName)
		if !ok {
			// In theory there's still a chance that cluster was just removed by another routine after the upper for-loop check.
			continue
		}
		c := v.(types.Cluster)
		c.StopHealthChecking()

		cm.clustersMap.Delete(clusterName)
		configmanager.SetRemoveClusterConfig(clusterName)
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[upstream] [cluster manager] Remove Primary Cluster, Cluster Name = %s", clusterName)
		}
	}
	return nil
}

// EDS Handler
func NewSimpleHostHandler(c types.Cluster, hostConfigs []v2.Host) {
	snap := c.Snapshot()
	hosts := make([]types.Host, 0, len(hostConfigs))
	for _, hc := range hostConfigs {
		hosts = append(hosts, NewSimpleHost(hc, snap.ClusterInfo()))
	}

	ns := NewHostSet(hosts)
	if snap.ClusterInfo().SlowStart().Mode != "" {
		transferHostSetStates(snap.HostSet(), ns)
	}

	c.UpdateHosts(ns)

}

func AppendSimpleHostHandler(c types.Cluster, hostConfigs []v2.Host) {
	snap := c.Snapshot()
	hosts := make([]types.Host, 0, len(hostConfigs))
	for _, hc := range hostConfigs {
		hosts = append(hosts, NewSimpleHost(hc, snap.ClusterInfo()))
	}
	snap.HostSet().Range(func(host types.Host) bool {
		hosts = append(hosts, host)
		return true
	})

	ns := NewHostSet(hosts)
	if snap.ClusterInfo().SlowStart().Mode != "" {
		transferHostSetStates(snap.HostSet(), ns)
	}

	c.UpdateHosts(ns)
}

// UpdateClusterHosts update all hosts in the cluster
func (cm *clusterManager) UpdateClusterHosts(clusterName string, hostConfigs []v2.Host) error {
	return cm.UpdateHosts(clusterName, hostConfigs, NewSimpleHostHandler)
}

// AppendClusterHosts adds new hosts into cluster
func (cm *clusterManager) AppendClusterHosts(clusterName string, hostConfigs []v2.Host) error {
	return cm.UpdateHosts(clusterName, hostConfigs, AppendSimpleHostHandler)
}

// RemoveClusterHosts removes hosts from cluster by address string
func (cm *clusterManager) RemoveClusterHosts(clusterName string, addrs []string) error {
	return cm.UpdateHosts(clusterName, nil,
		func(c types.Cluster, _ []v2.Host) {
			snap := c.Snapshot()
			newHosts := make([]types.Host, 0, snap.HostSet().Size())
			snap.HostSet().Range(func(host types.Host) bool {
				newHosts = append(newHosts, host)
				return true
			})
			sortedHosts := types.SortedHosts(newHosts)
			sort.Sort(sortedHosts)
			for _, addr := range addrs {
				i := sort.Search(sortedHosts.Len(), func(i int) bool {
					return sortedHosts[i].AddressString() >= addr
				})
				// found it, delete it
				if i < sortedHosts.Len() && sortedHosts[i].AddressString() == addr {
					sortedHosts = append(sortedHosts[:i], sortedHosts[i+1:]...)
				}
			}
			c.UpdateHosts(NewHostSet(sortedHosts))

		},
	)
}

func (cm *clusterManager) UpdateHosts(clusterName string, hostConfigs []v2.Host, hostHandler types.HostUpdateHandler) error {
	ci, ok := cm.clustersMap.Load(clusterName)
	if !ok {
		log.DefaultLogger.Errorf("[upstream] [cluster manager] cluster %s is not found", clusterName)
		return fmt.Errorf("cluster %s is not exists", clusterName)
	}
	c := ci.(types.Cluster)
	if hostHandler != nil {
		hostHandler(c, hostConfigs)
	}
	refreshHostsConfig(c)
	return nil
}

// GetClusterSnapshot returns cluster snap
func (cm *clusterManager) GetClusterSnapshot(ctx context.Context, clusterName string) types.ClusterSnapshot {
	ci, ok := cm.clustersMap.Load(clusterName)
	if !ok {
		return nil
	}
	return ci.(types.Cluster).Snapshot()
}

func (cm *clusterManager) PutClusterSnapshot(snap types.ClusterSnapshot) {
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

func (cm *clusterManager) UDPConnForCluster(lbCtx types.LoadBalancerContext, snapshot types.ClusterSnapshot) types.CreateConnectionData {
	if snapshot == nil || reflect.ValueOf(snapshot).IsNil() {
		return types.CreateConnectionData{}
	}
	host := snapshot.LoadBalancer().ChooseHost(lbCtx)
	if host == nil {
		return types.CreateConnectionData{}
	}
	return host.CreateUDPConnection(context.Background())
}

func (cm *clusterManager) ConnPoolForCluster(balancerContext types.LoadBalancerContext, snapshot types.ClusterSnapshot, protocol types.ProtocolName) (types.ConnectionPool, types.Host) {
	if snapshot == nil || reflect.ValueOf(snapshot).IsNil() {
		log.DefaultLogger.Errorf("[upstream] [cluster manager]  %s ConnPool For Cluster is nil", protocol)
		return nil, nil
	}
	pool, host, err := cm.getActiveConnectionPool(balancerContext, snapshot, protocol)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [cluster manager] ConnPoolForCluster Failed; %v", err)
	}
	return pool, host
}

func (cm *clusterManager) GetTLSManager() types.TLSClientContextManager {
	v := cm.tlsMng.Load()
	tlsmng, _ := v.(types.TLSClientContextManager)
	return tlsmng
}

const clusterManagerTLS = "mosn-cluster-manager-tls"

func (cm *clusterManager) UpdateTLSManager(tls *v2.TLSConfig) {
	if tls == nil {
		tls = &v2.TLSConfig{} // use a disabled config instead
	}
	mng, err := mtls.NewTLSClientContextManager(clusterManagerTLS, tls)
	if err != nil {
		log.DefaultLogger.Alertf("cluster.config", "[upstream] [cluster manager] NewClusterManager: Add TLS Manager failed, error: %v", err)
		return
	}
	cm.tlsMng.Store(mng)
	configmanager.SetClusterManagerTLS(*tls)
}

const (
	maxHostsCounts  = 3
	intervalStep    = 5
	maxTryConnTimes = 15
)

var tryConnTimes = [maxTryConnTimes]time.Duration{}

func init() {
	index := 0
	// total duration is 535ms
	for _, interval := range []time.Duration{
		2 * time.Millisecond,
		5 * time.Millisecond,
		100 * time.Millisecond,
	} {
		for i := 0; i < intervalStep; i++ {
			tryConnTimes[index] = interval
			index++
		}
	}
}

var (
	errNilHostChoose   = errors.New("cluster snapshot choose host is nil")
	errUnknownProtocol = errors.New("protocol pool can not found protocol")
	errNoHealthyHost   = errors.New("no health hosts")
)

func (cm *clusterManager) getActiveConnectionPool(balancerContext types.LoadBalancerContext, clusterSnapshot types.ClusterSnapshot, proto types.ProtocolName) (types.ConnectionPool, types.Host, error) {
	factory, ok := protocol.GetNewPoolFactory(proto)
	if !ok {
		return nil, nil, fmt.Errorf("protocol %v is not registered in pool factory", proto)
	}

	// for pool to know, whether this is a multiplex or pingpong pool
	var (
		pools [maxHostsCounts]types.ConnectionPool
		hosts [maxHostsCounts]types.Host
	)

	try := clusterSnapshot.HostNum(balancerContext.MetadataMatchCriteria())
	if try == 0 {
		return nil, nil, errNilHostChoose
	}
	if try > maxHostsCounts {
		try = maxHostsCounts
	}
	for i := 0; i < try; i++ {
		host := clusterSnapshot.LoadBalancer().ChooseHost(balancerContext)
		if host == nil {
			return nil, nil, errNilHostChoose
		}

		addr := host.AddressString()
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[upstream] [cluster manager] clusterSnapshot.loadbalancer.ChooseHost result is %s, cluster name = %s", addr, clusterSnapshot.ClusterInfo().Name())
		}
		connectionPool, ok := cm.protocolConnPool.load(proto, clusterSnapshot)
		if !ok {
			return nil, nil, errUnknownProtocol
		}
		// we cannot use sync.Map.LoadOrStore directly, because we do not want to new a connpool every time
		loadOrStoreConnPool := func() (types.ConnectionPool, bool) {
			// avoid locking if it is already exists
			if connPool, ok := connectionPool.Load(addr); ok {
				pool := connPool.(types.ConnectionPool)
				return pool, true
			}
			cm.mux.Lock()
			defer cm.mux.Unlock()
			if connPool, ok := connectionPool.Load(addr); ok {
				pool := connPool.(types.ConnectionPool)
				return pool, true
			}
			pool := factory(balancerContext.DownstreamContext(), host)
			connectionPool.Store(addr, pool)
			return pool, false
		}
		pool, loaded := loadOrStoreConnPool()
		if loaded {
			if !pool.TLSHashValue().Equal(host.TLSHashValue()) {
				if log.DefaultLogger.GetLogLevel() >= log.INFO {
					log.DefaultLogger.Infof("[upstream] [cluster manager] %s tls state changed", addr)
				}
				func() {
					// lock the load and delete
					cm.mux.Lock()
					defer cm.mux.Unlock()
					// recheck whether the pool is changed
					if connPool, ok := connectionPool.Load(addr); ok {
						pool = connPool.(types.ConnectionPool)
						if pool.TLSHashValue().Equal(host.TLSHashValue()) {
							return
						}
						connectionPool.Delete(addr)
						pool.Shutdown()
						pool = factory(balancerContext.DownstreamContext(), host)
						connectionPool.Store(addr, pool)
						cm.tlsMetrics.TLSConnpoolChanged.Inc(1)
					}
				}()

			}
		}
		if pool.CheckAndInit(balancerContext.DownstreamContext()) {
			return pool, host, nil
		}
		pools[i] = pool
		hosts[i] = host
	}

	// perhaps the first request, wait for tcp handshaking.
	for t := 0; t < maxTryConnTimes; t++ {
		for i := 0; i < try; i++ {
			if pools[i] == nil {
				continue
			}
			if pools[i].CheckAndInit(balancerContext.DownstreamContext()) {
				return pools[i], hosts[i], nil
			}
		}
		waitTime := tryConnTimes[t]
		time.Sleep(waitTime)
	}
	return nil, nil, errNoHealthyHost
}

func (cm *clusterManager) ShutdownConnectionPool(proto types.ProtocolName, addr string) {
	cm.protocolConnPool.shutdown(proto, addr)
}
