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
	"net"

	"github.com/alipay/sofa-mosn/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	proto "github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/stream/http"
	"github.com/alipay/sofa-mosn/pkg/stream/http2"
	"github.com/alipay/sofa-mosn/pkg/stream/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/stream/xprotocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/orcaman/concurrent-map"
)

// ClusterManager
type clusterManager struct {
	sourceAddr             net.Addr
	primaryClusters        cmap.ConcurrentMap // string: *primaryCluster
	sofaRPCConnPool        cmap.ConcurrentMap // string: types.ConnectionPool
	http2ConnPool          cmap.ConcurrentMap // string: types.ConnectionPool
	xProtocolConnPool      cmap.ConcurrentMap // string: types.ConnectionPool
	http1ConnPool          cmap.ConcurrentMap // string: types.ConnectionPool
	clusterAdapter         Adapter
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
	cm := &clusterManager{
		sourceAddr:        sourceAddr,
		primaryClusters:   cmap.New(),
		sofaRPCConnPool:   cmap.New(),
		http2ConnPool:     cmap.New(),
		xProtocolConnPool: cmap.New(),
		http1ConnPool:     cmap.New(),
		autoDiscovery:     true, //todo delete
	}
	//init Adap when run app
	Adap = Adapter{
		clusterMng: cm,
	}

	cm.clusterAdapter = Adap

	//Add cluster to cm
	//Register upstream update type
	for _, cluster := range clusters {
		cm.AddOrUpdatePrimaryCluster(cluster)
	}

	// Add hosts to cluster
	// Note: currently, use priority = 0
	for clusterName, hosts := range clusterMap {
		cm.UpdateClusterHosts(clusterName, 0, hosts)
	}

	return cm
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
}

func (cm *clusterManager) AddOrUpdatePrimaryCluster(cluster v2.Cluster) bool {
	clusterName := cluster.Name

	if v, exist := cm.primaryClusters.Get(clusterName); exist {
		if !v.(*primaryCluster).addedViaAPI {
			return false
		}
	}

	// todo for static cluster, shouldn't use this way
	cm.loadCluster(cluster, true)

	return true
}

func (cm *clusterManager) ClusterExist(clusterName string) bool {
	if _, exist := cm.primaryClusters.Get(clusterName); exist {
		return true
	}

	return false
}

func (cm *clusterManager) loadCluster(clusterConfig v2.Cluster, addedViaAPI bool) types.Cluster {
	//clusterConfig.UseHealthCheck
	cluster := NewCluster(clusterConfig, cm.sourceAddr, addedViaAPI)

	cluster.Initialize(func() {
		cluster.PrioritySet().AddMemberUpdateCb(func(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
		})
	})

	cm.primaryClusters.Set(clusterConfig.Name, &primaryCluster{
		cluster:     cluster,
		addedViaAPI: addedViaAPI,
	})

	return cluster
}

func (cm *clusterManager) getOrCreateClusterSnapshot(clusterName string) *clusterSnapshot {
	if v, ok := cm.primaryClusters.Get(clusterName); ok {
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

func (cm *clusterManager) SetInitializedCb(cb func()) {}

func (cm *clusterManager) Clusters() map[string]types.Cluster {
	clusterInfoMap := make(map[string]types.Cluster)

	for c, pc := range cm.primaryClusters.Items() {
		clusterInfoMap[c] = pc.(*primaryCluster).cluster
	}

	return clusterInfoMap
}

func (cm *clusterManager) Get(context context.Context, cluster string) types.ClusterSnapshot {
	return cm.getOrCreateClusterSnapshot(cluster)
}

func (cm *clusterManager) UpdateClusterHosts(clusterName string, priority uint32, hostConfigs []v2.Host) error {
	if v, ok := cm.primaryClusters.Get(clusterName); ok {
		pcc := v.(*primaryCluster).cluster

		// todo: hack
		if concretedCluster, ok := pcc.(*simpleInMemCluster); ok {
			var hosts []types.Host

			for _, hc := range hostConfigs {
				hosts = append(hosts, NewHost(hc, pcc.Info()))
			}
			concretedCluster.UpdateHosts(hosts)
			return nil
		}
		return fmt.Errorf("cluster's hostset %s can't be update", clusterName)

	}
	return fmt.Errorf("cluster %s not found", clusterName)

}

func (cm *clusterManager) RemoveClusterHosts(clusterName string, host types.Host) error {
	if host == nil {
		return errors.New("host is nil")
	}

	if v, ok := cm.primaryClusters.Get(clusterName); ok {
		pcc := v.(*primaryCluster).cluster

		found := false
		if concretedCluster, ok := pcc.(*simpleInMemCluster); ok {
			ccHosts := concretedCluster.hosts
			for i := 0; i < len(ccHosts); i++ {

				if host.AddressString() == ccHosts[i].AddressString() {
					ccHosts = append(ccHosts[:i], ccHosts[i+1:]...)
					found = true
					break
				}
			}
			if found == true {
				log.DefaultLogger.Debugf("Remove Host Success, Host Address is %s", host.AddressString())
				concretedCluster.UpdateHosts(ccHosts)
			} else {
				log.DefaultLogger.Debugf("Remove Host Failed, Host %s Doesn't Exist", host.AddressString())

			}

		} else {
			return fmt.Errorf("cluster's hostset %s can't be update", clusterName)
		}
	}

	return nil
}

func (cm *clusterManager) HTTPConnPoolForCluster(lbCtx types.LoadBalancerContext, cluster string,
	protocol types.Protocol) types.ConnectionPool {
	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster)

	if clusterSnapshot == nil {
		return nil
	}

	host := clusterSnapshot.loadbalancer.ChooseHost(lbCtx)

	if host != nil {
		addr := host.AddressString()
		log.StartLogger.Tracef("http connection pool upstream addr : %v", addr)

		switch protocol {
		case proto.HTTP2:

			if connPool, ok := cm.http2ConnPool.Get(addr); ok {
				return connPool.(types.ConnectionPool)
			}
			// todo: move this to a centralized factory, remove dependency to http2 stream
			connPool := http2.NewConnPool(host)
			cm.http2ConnPool.Set(addr, connPool)

			return connPool
		case proto.HTTP1:

			if connPool, ok := cm.http1ConnPool.Get(addr); ok {
				return connPool.(types.ConnectionPool)
			}
			// todo: move this to a centralized factory, remove dependency to http1 stream
			connPool := http.NewConnPool(host)
			cm.http1ConnPool.Set(addr, connPool)

			return connPool
		}

	}

	return nil
}

func (cm *clusterManager) XprotocolConnPoolForCluster(lbCtx types.LoadBalancerContext, cluster string,
	protocol types.Protocol) types.ConnectionPool {
	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster)

	if clusterSnapshot == nil {
		return nil
	}

	host := clusterSnapshot.loadbalancer.ChooseHost(nil)

	if host != nil {
		addr := host.AddressString()
		log.StartLogger.Tracef("Xprotocol connection pool upstream addr : %v", addr)

		if connPool, ok := cm.xProtocolConnPool.Get(addr); ok {
			return connPool.(types.ConnectionPool)
		}
		connPool := xprotocol.NewConnPool(host)
		cm.xProtocolConnPool.Set(addr, connPool)

		return connPool
	}

	return nil
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

func (cm *clusterManager) SofaRPCConnPoolForCluster(lbCtx types.LoadBalancerContext, cluster string) types.ConnectionPool {
	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster)

	if clusterSnapshot == nil {
		log.DefaultLogger.Errorf(" Sofa Rpc ConnPool For Cluster is nil, cluster name = %s", cluster)
		return nil
	}

	host := clusterSnapshot.loadbalancer.ChooseHost(lbCtx)

	if host != nil {
		addr := host.AddressString()
		log.DefaultLogger.Debugf(" clusterSnapshot.loadbalancer.ChooseHost result is %s, cluster name = %s", addr, cluster)

		if connPool, ok := cm.sofaRPCConnPool.Get(addr); ok {
			return connPool.(types.ConnectionPool)
		}
		// todo: move this to a centralized factory, remove dependency to sofarpc stream
		connPool := sofarpc.NewConnPool(host)
		cm.sofaRPCConnPool.Set(addr, connPool)

		return connPool

	}

	log.DefaultLogger.Errorf("clusterSnapshot.loadbalancer.ChooseHost is nil, cluster name = %s", cluster)
	return nil
}

func (cm *clusterManager) RemovePrimaryCluster(clusterName string) bool {
	if v, exist := cm.primaryClusters.Get(clusterName); exist {
		if !v.(*primaryCluster).addedViaAPI {
			return false
			log.DefaultLogger.Warnf("Remove Primary Cluster Failed, Cluster Name = %s not addedViaAPI", clusterName)
		} else {
			cm.primaryClusters.Remove(clusterName)
			log.DefaultLogger.Debugf("Remove Primary Cluster, Cluster Name = %s", clusterName)
		}
	}

	return true
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
