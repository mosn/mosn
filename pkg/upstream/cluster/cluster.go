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
	"net"
	"sync"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mtls"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// Cluster
type cluster struct {
	initializationStarted          bool
	initializationCompleteCallback func()
	prioritySet                    *prioritySet
	info                           *clusterInfo
	mux                            sync.RWMutex
	initHelper                     concreteClusterInitHelper
	healthChecker                  types.HealthChecker
}

type concreteClusterInitHelper interface {
	Init()
}

func NewCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaAPI bool) types.Cluster {
	var newCluster types.Cluster

	switch clusterConfig.ClusterType {

	case v2.SIMPLE_CLUSTER, v2.DYNAMIC_CLUSTER, v2.EDS_CLUSTER:
		newCluster = newSimpleInMemCluster(clusterConfig, sourceAddr, addedViaAPI)
	default:
		return nil
	}

	// init health check for cluster's host
	if clusterConfig.HealthCheck.Protocol != "" {
		var hc types.HealthChecker
		hc = types.HealthCheckFactoryInstance.New(clusterConfig.HealthCheck)
		newCluster.SetHealthChecker(hc)
	}

	return newCluster
}

func newCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaAPI bool, initHelper concreteClusterInitHelper) *cluster {
	cluster := cluster{
		prioritySet: &prioritySet{},
		info: &clusterInfo{
			name:                 clusterConfig.Name,
			clusterType:          clusterConfig.ClusterType,
			sourceAddr:           sourceAddr,
			addedViaAPI:          addedViaAPI,
			maxRequestsPerConn:   clusterConfig.MaxRequestPerConn,
			connBufferLimitBytes: clusterConfig.ConnBufferLimitBytes,
			stats:                newClusterStats(clusterConfig.Name),
			lbSubsetInfo:         NewLBSubsetInfo(&clusterConfig.LBSubSetConfig), // new subset load balancer info
		},
		initHelper: initHelper,
	}

	switch clusterConfig.LbType {
	case v2.LB_RANDOM:
		cluster.info.lbType = types.Random

	case v2.LB_ROUNDROBIN:
		cluster.info.lbType = types.RoundRobin
	}

	// TODO: init more props: maxrequestsperconn, connecttimeout, connectionbuflimit

	cluster.info.resourceManager = NewResourceManager(clusterConfig.CirBreThresholds)

	cluster.prioritySet.GetOrCreateHostSet(0)
	cluster.prioritySet.AddMemberUpdateCb(func(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
		// TODO: update cluster stats
	})

	var lb types.LoadBalancer

	if cluster.Info().LbSubsetInfo().IsEnabled() {
		// use subset loadbalancer
		lb = NewSubsetLoadBalancer(cluster.Info().LbType(), cluster.PrioritySet(), cluster.Info().Stats(),
			cluster.Info().LbSubsetInfo())

	} else {
		// use common loadbalancer
		lb = NewLoadBalancer(cluster.Info().LbType(), cluster.PrioritySet())
	}

	cluster.info.lbInstance = lb

	mgr, err := mtls.NewTLSClientContextManager(&clusterConfig.TLS, cluster.info)
	if err != nil {
		log.DefaultLogger.Fatalf("create tls context manager failed, %v", err)
	}
	cluster.info.tlsMng = mgr

	return &cluster
}

func (c *cluster) Initialize(cb func()) {
	c.initializationCompleteCallback = cb

	if c.initHelper != nil {
		c.initHelper.Init()
	}

	if c.initializationCompleteCallback != nil {
		c.initializationCompleteCallback()
	}
}

func (c *cluster) Info() types.ClusterInfo {
	return c.info
}

func (c *cluster) InitializePhase() types.InitializePhase {
	return types.Primary
}

func (c *cluster) PrioritySet() types.PrioritySet {
	return c.prioritySet
}

func (c *cluster) SetHealthChecker(hc types.HealthChecker) {
	c.healthChecker = hc
	c.healthChecker.SetCluster(c)
	c.healthChecker.Start()
	c.healthChecker.AddHostCheckCompleteCb(func(host types.Host, changedState bool) {
		if changedState {
			c.refreshHealthHosts(host)
		}
	})
}

func (c *cluster) HealthChecker() types.HealthChecker {
	return c.healthChecker
}

// update health-hostSet for only one hostSet, reduce update times
func (c *cluster) refreshHealthHosts(host types.Host) {
	if host.Health() {
		log.DefaultLogger.Debugf("Add health host %s to cluster's healthHostSet by refreshHealthHosts", host.AddressString())
		addHealthyHost(c.prioritySet.hostSets, host)
	} else {
		log.DefaultLogger.Debugf("Del host %s from cluster's healthHostSet by refreshHealthHosts", host.AddressString())
		delHealthHost(c.prioritySet.hostSets, host)
	}
}

// refresh health hosts globally
func (c *cluster) refreshHealthHostsGlobal() {

	for _, hostSet := range c.prioritySet.hostSets {
		var healthyHost []types.Host
		var healthyHostPerLocality [][]types.Host

		healthyHost = getHealthHost(hostSet.Hosts())
		healthyHostPerLocality = getHealthHostsPerLocality(hostSet.HostsPerLocality())

		hostSet.UpdateHosts(hostSet.Hosts(), healthyHost, hostSet.HostsPerLocality(),
			healthyHostPerLocality, nil, nil)
	}
}

type clusterInfo struct {
	name                 string
	clusterType          v2.ClusterType
	lbType               types.LoadBalancerType // if use subset lb , lbType is used as inner LB algorithm for choosing subset's host
	lbInstance           types.LoadBalancer     // load balancer used for this cluster
	sourceAddr           net.Addr
	connectTimeout       int
	connBufferLimitBytes uint32
	features             int
	maxRequestsPerConn   uint32
	addedViaAPI          bool
	resourceManager      types.ResourceManager
	stats                types.ClusterStats
	healthCheckProtocol  string
	tlsMng               types.TLSContextManager
	lbSubsetInfo         types.LBSubsetInfo
}

func NewClusterInfo() types.ClusterInfo {
	return &clusterInfo{}
}

func (ci *clusterInfo) Name() string {
	return ci.name
}

func (ci *clusterInfo) LbType() types.LoadBalancerType {
	return ci.lbType
}

func (ci *clusterInfo) AddedViaAPI() bool {
	return ci.addedViaAPI
}

func (ci *clusterInfo) SourceAddress() net.Addr {
	return ci.sourceAddr
}

func (ci *clusterInfo) ConnectTimeout() int {
	return ci.connectTimeout
}

func (ci *clusterInfo) ConnBufferLimitBytes() uint32 {
	return ci.connBufferLimitBytes
}

func (ci *clusterInfo) Features() int {
	return ci.features
}

func (ci *clusterInfo) Metadata() v2.Metadata {
	return v2.Metadata{}
}

func (ci *clusterInfo) DiscoverType() string {
	return ""
}

func (ci *clusterInfo) MaintenanceMode() bool {
	return false
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

func (ci *clusterInfo) HealthCheckProtocol() string {
	return ci.healthCheckProtocol
}

func (ci *clusterInfo) TLSMng() types.TLSContextManager {
	return ci.tlsMng
}

func (ci *clusterInfo) LbSubsetInfo() types.LBSubsetInfo {
	return ci.lbSubsetInfo
}

func (ci *clusterInfo) LBInstance() types.LoadBalancer {
	return ci.lbInstance
}

type prioritySet struct {
	hostSets        []types.HostSet // Note: index is the priority
	updateCallbacks []types.MemberUpdateCallback
	mux             sync.RWMutex
}

func (ps *prioritySet) GetOrCreateHostSet(priority uint32) types.HostSet {
	ps.mux.Lock()
	defer ps.mux.Unlock()

	// Create a priority set
	if uint32(len(ps.hostSets)) < priority+1 {

		for i := uint32(len(ps.hostSets)); i <= priority; i++ {
			hostSet := ps.createHostSet(i)
			hostSet.addMemberUpdateCb(func(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
				for _, cb := range ps.updateCallbacks {
					cb(priority, hostsAdded, hostsRemoved)
				}
			})
			ps.hostSets = append(ps.hostSets, hostSet)
		}
	}

	return ps.hostSets[priority]
}

func (ps *prioritySet) createHostSet(priority uint32) *hostSet {
	return &hostSet{
		priority: priority,
	}
}

func (ps *prioritySet) GetHostsInfo(priority uint32) []types.HostInfo {
	var hostinfos []types.HostInfo
	if uint32(len(ps.hostSets)) > priority {
		hostset := ps.hostSets[priority]
		for _, host := range hostset.Hosts() {
			// host is an implement of hostinfo
			hostinfos = append(hostinfos, host)
		}
	}
	return hostinfos

}

func (ps *prioritySet) AddMemberUpdateCb(cb types.MemberUpdateCallback) {
	ps.updateCallbacks = append(ps.updateCallbacks, cb)
}

func (ps *prioritySet) HostSetsByPriority() []types.HostSet {
	ps.mux.RLock()
	defer ps.mux.RUnlock()

	return ps.hostSets
}

func getHealthHost(hosts []types.Host) []types.Host {
	var healthyHost []types.Host
	// todo: calculate healthyHost & healthyHostPerLocality
	for _, h := range hosts {
		if h.Health() {
			healthyHost = append(healthyHost, h)
		}
	}
	return healthyHost
}

func getHealthHostsPerLocality(hhpl [][]types.Host) [][]types.Host {
	var healthyHostPerLocality = make([][]types.Host, len(hhpl))
	hi := 0

	for i := range hhpl {
		hj := 0
		for j := range hhpl[i] {
			if hhpl[i][j].Health() {
				healthyHostPerLocality[hi] = append(healthyHostPerLocality[hi], hhpl[i][j])
				hj++
			}
		}
		if hj > 0 {
			hi++
		}
	}
	return healthyHostPerLocality
}

func addHealthyHost(hostSets []types.HostSet, host types.Host) {
	// Note: currently, one host only belong to a hostSet

	for i, hostSet := range hostSets {
		found := false

		for _, h := range hostSet.Hosts() {
			if h.AddressString() == host.AddressString() {
				log.DefaultLogger.Debugf("add healthy host = %s, in priority = %d", host.AddressString(), i)
				found = true
				break
			}
		}

		if found {
			newHealthHost := hostSet.HealthyHosts()
			newHealthHost = append(newHealthHost, host)
			newHealthyHostPerLocality := hostSet.HealthHostsPerLocality()
			newHealthyHostPerLocality[len(newHealthyHostPerLocality)-1] = append(newHealthyHostPerLocality[len(newHealthyHostPerLocality)-1], host)

			hostSet.UpdateHosts(hostSet.Hosts(), newHealthHost, hostSet.HostsPerLocality(),
				newHealthyHostPerLocality, nil, nil)
			break
		}
	}
}

func delHealthHost(hostSets []types.HostSet, host types.Host) {
	for i, hostSet := range hostSets {
		// Note: currently, one host only belong to a hostSet
		found := false

		for _, h := range hostSet.Hosts() {
			if h.AddressString() == host.AddressString() {
				log.DefaultLogger.Debugf("del healthy host = %s, in priority = %d", host.AddressString(), i)
				found = true
				break
			}
		}

		if found {
			newHealthHost := hostSet.HealthyHosts()
			newHealthyHostPerLocality := hostSet.HealthHostsPerLocality()

			for i, hh := range newHealthHost {
				if host.Hostname() == hh.Hostname() {
					//remove
					newHealthHost = append(newHealthHost[:i], newHealthHost[i+1:]...)
					break
				}
			}

			for i := range newHealthyHostPerLocality {
				for j := range newHealthyHostPerLocality[i] {

					if host.Hostname() == newHealthyHostPerLocality[i][j].Hostname() {
						newHealthyHostPerLocality[i] = append(newHealthyHostPerLocality[i][:j], newHealthyHostPerLocality[i][j+1:]...)
						break
					}
				}
			}

			hostSet.UpdateHosts(hostSet.Hosts(), newHealthHost, hostSet.HostsPerLocality(),
				newHealthyHostPerLocality, nil, nil)
			break
		}
	}
}
