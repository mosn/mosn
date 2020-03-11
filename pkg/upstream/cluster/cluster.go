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
	"sync/atomic"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/healthcheck"
	"mosn.io/pkg/utils"
)

func NewCluster(clusterConfig v2.Cluster) types.Cluster {
	// TODO: support cluster type registered
	return newSimpleCluster(clusterConfig)
}

// simpleCluster is an implementation of types.Cluster
type simpleCluster struct {
	info          *clusterInfo
	healthChecker types.HealthChecker
	lbInstance    types.LoadBalancer // load balancer used for this cluster
	hostSet       *hostSet
	snapshot      atomic.Value
}

func newSimpleCluster(clusterConfig v2.Cluster) *simpleCluster {
	info := &clusterInfo{
		name:                 clusterConfig.Name,
		clusterType:          clusterConfig.ClusterType,
		maxRequestsPerConn:   clusterConfig.MaxRequestPerConn,
		connBufferLimitBytes: clusterConfig.ConnBufferLimitBytes,
		stats:                newClusterStats(clusterConfig.Name),
		lbSubsetInfo:         NewLBSubsetInfo(&clusterConfig.LBSubSetConfig), // new subset load balancer info
		lbOriDstInfo:         NewLBOriDstInfo(&clusterConfig.LBOriDstConfig), // new oridst load balancer info
		lbType:               types.LoadBalancerType(clusterConfig.LbType),
		resourceManager:      NewResourceManager(clusterConfig.CirBreThresholds),
	}

	// set ConnectTimeout
	if clusterConfig.ConnectTimeout != nil {
		info.connectTimeout = clusterConfig.ConnectTimeout.Duration
	} else {
		info.connectTimeout = network.DefaultConnectTimeout
	}

	// tls mng
	mgr, err := mtls.NewTLSClientContextManager(&clusterConfig.TLS)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [cluster] [new cluster] create tls context manager failed, %v", err)
	}
	info.tlsMng = mgr
	cluster := &simpleCluster{
		info: info,
	}
	// init a empty
	hostSet := &hostSet{}
	cluster.snapshot.Store(&clusterSnapshot{
		info:    info,
		hostSet: hostSet,
		lb:      NewLoadBalancer(info.lbType, hostSet),
	})
	if clusterConfig.HealthCheck.ServiceName != "" {
		log.DefaultLogger.Infof("[upstream] [cluster] [new cluster] cluster %s have health check", clusterConfig.Name)
		cluster.healthChecker = healthcheck.CreateHealthCheck(clusterConfig.HealthCheck)
		cluster.healthChecker.AddHostCheckCompleteCb(func(host types.Host, changedState bool, isHealthy bool) {
			if changedState {
				log.DefaultLogger.Infof("[upstream] [cluster] host %s state change to %v", host.AddressString(), isHealthy)
				cluster.hostSet.refreshHealthHost(host)
			}
		})

	}
	return cluster
}

func (sc *simpleCluster) UpdateHosts(newHosts []types.Host) {
	info := sc.info
	hostSet := &hostSet{}
	hostSet.setFinalHost(newHosts)
	// load balance
	var lb types.LoadBalancer
	if info.lbSubsetInfo.IsEnabled() {
		lb = NewSubsetLoadBalancer(info, hostSet)
	} else {
		lb = NewLoadBalancer(info.lbType, hostSet)
	}
	sc.lbInstance = lb
	sc.hostSet = hostSet
	sc.snapshot.Store(&clusterSnapshot{
		lb:      lb,
		hostSet: hostSet,
		info:    info,
	})
	if sc.healthChecker != nil {
		utils.GoWithRecover(func() {
			sc.healthChecker.SetHealthCheckerHostSet(hostSet)
		}, nil)
	}

}

func (sc *simpleCluster) Snapshot() types.ClusterSnapshot {
	si := sc.snapshot.Load()
	if snap, ok := si.(*clusterSnapshot); ok {
		return snap
	}
	return nil
}

func (sc *simpleCluster) AddHealthCheckCallbacks(cb types.HealthCheckCb) {
	if sc.healthChecker != nil {
		sc.healthChecker.AddHostCheckCompleteCb(cb)
	}
}

func (sc *simpleCluster) StopHealthChecking() {
	if sc.healthChecker != nil {
		sc.healthChecker.Stop()
	}
}

type clusterInfo struct {
	name                 string
	clusterType          v2.ClusterType
	lbType               types.LoadBalancerType // if use subset lb , lbType is used as inner LB algorithm for choosing subset's host
	connBufferLimitBytes uint32
	maxRequestsPerConn   uint32
	resourceManager      types.ResourceManager
	stats                types.ClusterStats
	lbSubsetInfo         types.LBSubsetInfo
	lbOriDstInfo         types.LBOriDstInfo
	tlsMng               types.TLSContextManager
	connectTimeout       time.Duration
}

func (ci *clusterInfo) Name() string {
	return ci.name
}

func (ci *clusterInfo) ClusterType() v2.ClusterType {
	return ci.clusterType
}

func (ci *clusterInfo) LbType() types.LoadBalancerType {
	return ci.lbType
}

func (ci *clusterInfo) ConnBufferLimitBytes() uint32 {
	return ci.connBufferLimitBytes
}

func (ci *clusterInfo) MaxRequestsPerConn() uint32 {
	return ci.maxRequestsPerConn
}

func (ci *clusterInfo) Stats() types.ClusterStats {
	return ci.stats
}

func (ci *clusterInfo) ResourceManager() types.ResourceManager {
	return ci.resourceManager
}

func (ci *clusterInfo) TLSMng() types.TLSContextManager {
	return ci.tlsMng
}

func (ci *clusterInfo) LbSubsetInfo() types.LBSubsetInfo {
	return ci.lbSubsetInfo
}

func (ci *clusterInfo) ConnectTimeout() time.Duration {
	return ci.connectTimeout
}

func (ci *clusterInfo) LbOriDstInfo() types.LBOriDstInfo {
	return ci.lbOriDstInfo
}

type clusterSnapshot struct {
	info    types.ClusterInfo
	hostSet types.HostSet
	lb      types.LoadBalancer
}

func (snapshot *clusterSnapshot) HostSet() types.HostSet {
	return snapshot.hostSet
}

func (snapshot *clusterSnapshot) ClusterInfo() types.ClusterInfo {
	return snapshot.info
}

func (snapshot *clusterSnapshot) LoadBalancer() types.LoadBalancer {
	return snapshot.lb
}

func (snapshot *clusterSnapshot) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return snapshot.lb.IsExistsHosts(metadata)
}

func (snapshot *clusterSnapshot) HostNum(metadata api.MetadataMatchCriteria) int {
	return snapshot.lb.HostNum(metadata)
}
