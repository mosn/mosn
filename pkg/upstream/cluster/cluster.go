package cluster

import (
	"net"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"github.com/rcrowley/go-metrics"
)

// Cluster
type cluster struct {
	initializationStarted          bool
	initializationCompleteCallback func()
	prioritySet                    *prioritySet
	info                           *clusterInfo
	initHelper                     concreteClusterInitHelper
}

type concreteClusterInitHelper interface {
	Init()
}

func NewCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaApi bool) types.Cluster {
	var newCluster types.Cluster

	switch clusterConfig.ClusterType {
	case v2.SIMPLE_CLUSTER:
		newCluster = newSimpleInMemCluster(clusterConfig, sourceAddr, addedViaApi)
	}

	return newCluster
}

func newCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaApi bool, initHelper concreteClusterInitHelper) cluster {
	cluster := cluster{
		prioritySet: &prioritySet{},
		info: &clusterInfo{
			name:                 clusterConfig.Name,
			clusterType:          clusterConfig.ClusterType,
			sourceAddr:           sourceAddr,
			addedViaApi:          addedViaApi,
			connBufferLimitBytes: clusterConfig.ConnBufferLimitBytes,
			stats:                newClusterStats(clusterConfig),
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

	// TODO: change hardcode to read from config @wugou
	cluster.info.resourceManager = NewResourceManager(102400, 102400, 102400)

	cluster.prioritySet.GetOrCreateHostSet(0)
	cluster.prioritySet.AddMemberUpdateCb(func(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
		// TODO: update cluster stats
	})

	return cluster
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

func newClusterStats(config v2.Cluster) types.ClusterStats {
	nameSpace := fmt.Sprintf("cluster.%s", config.Name)

	return types.ClusterStats{
		Namespace:                                      nameSpace,
		UpstreamConnectionTotal:                        metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_total"), nil),
		UpstreamConnectionClose:                        metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_close"), nil),
		UpstreamConnectionActive:                       metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_active"), nil),
		UpstreamConnectionTotalHttp1:                   metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_total_http1"), nil),
		UpstreamConnectionTotalHttp2:                   metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_total_http2"), nil),
		UpstreamConnectionTotalSofaRpc:                 metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_total_sofarpc"), nil),
		UpstreamConnectionConFail:                      metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_con_fail"), nil),
		UpstreamConnectionRetry:                        metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_retry"), nil),
		UpstreamConnectionLocalClose:                   metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_local_close"), nil),
		UpstreamConnectionRemoteClose:                  metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_remote_close"), nil),
		UpstreamConnectionLocalCloseWithActiveRequest:  metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_local_close_with_active_request"), nil),
		UpstreamConnectionRemoteCloseWithActiveRequest: metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_remote_close_with_active_request"), nil),
		UpstreamConnectionCloseNotify:                  metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_close_notify"), nil),
		UpstreamBytesRead:                              metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_bytes_read"), nil),
		UpstreamBytesReadCurrent:                       metrics.GetOrRegisterGauge(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_bytes_read_current"), nil),
		UpstreamBytesWrite:                             metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_bytes_write"), nil),
		UpstreamBytesWriteCurrent:                      metrics.GetOrRegisterGauge(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_bytes_write_current"), nil),
		UpstreamRequestTotal:                           metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_request_total"), nil),
		UpstreamRequestActive:                          metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_request_active"), nil),
		UpstreamRequestLocalReset:                      metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_request_local_reset"), nil),
		UpstreamRequestRemoteReset:                     metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_request_remote_reset"), nil),
		UpstreamRequestTimeout:                         metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_request_timeout"), nil),
		UpstreamRequestFailureEject:                    metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_failure_eject"), nil),
		UpstreamRequestPendingOverflow:                 metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_pending_overflow"), nil),
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

func (c *cluster) HealthChecker() types.HealthChecker {
	// TODO
	return nil
}

func (c *cluster) OutlierDetector() types.Detector {
	// TODO
	return nil
}

type clusterInfo struct {
	name                 string
	clusterType          v2.ClusterType
	lbType               types.LoadBalancerType
	sourceAddr           net.Addr
	connectTimeout       int
	connBufferLimitBytes uint32
	features             int
	maxRequestsPerConn   uint64
	addedViaApi          bool
	resourceManager      types.ResourceManager
	stats                types.ClusterStats
}

func (ci *clusterInfo) Name() string {
	return ci.name
}

func (ci *clusterInfo) LbType() types.LoadBalancerType {
	return ci.lbType
}

func (ci *clusterInfo) AddedViaApi() bool {
	return ci.addedViaApi
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

func (ci *clusterInfo) MaxRequestsPerConn() uint64 {
	return ci.maxRequestsPerConn
}

func (ci *clusterInfo) Stats() types.ClusterStats {
	return ci.stats
}

func (ci *clusterInfo) ResourceManager() types.ResourceManager {
	return ci.resourceManager
}

type prioritySet struct {
	hostSets        []types.HostSet
	updateCallbacks []types.MemberUpdateCallback
}

func (ps *prioritySet) GetOrCreateHostSet(priority uint32) types.HostSet {
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

func (ps *prioritySet) AddMemberUpdateCb(cb types.MemberUpdateCallback) {
	ps.updateCallbacks = append(ps.updateCallbacks, cb)
}

func (ps *prioritySet) HostSetsByPriority() []types.HostSet {
	return ps.hostSets
}

type dynamicClusterbase struct {
	cluster
}

func (dc *dynamicClusterbase) updateDynamicHostList(newHosts []types.Host, currentHosts []types.Host) (
	changed bool, finalHosts []types.Host, hostsAdded []types.Host, hostsRemoved []types.Host) {
	hostAddrs := make(map[string]bool)

	// N^2 loop, works for small and steady hosts
	for _, nh := range newHosts {
		nhAddr := nh.Address().String()
		if _, ok := hostAddrs[nhAddr]; ok {
			continue
		}

		hostAddrs[nhAddr] = true

		found := false
		for i := 0; i < len(currentHosts); {
			curNh := currentHosts[i]

			if nh.Address().String() == curNh.Address().String() {
				curNh.SetWeight(nh.Weight())
				finalHosts = append(finalHosts, curNh)
				currentHosts = append(currentHosts[:i], currentHosts[i+1:]...)
				found = true
			} else {
				i++
			}
		}

		if !found {
			finalHosts = append(finalHosts, nh)
			hostsAdded = append(hostsAdded, nh)
		}
	}

	if len(currentHosts) > 0 {
		hostsRemoved = currentHosts
	}

	if len(hostsAdded) > 0 || len(hostsRemoved) > 0 {
		changed = true
	} else {
		changed = false
	}

	return changed, finalHosts, hostsAdded, hostsRemoved
}

// SimpleCluster
type simpleInMemCluster struct {
	dynamicClusterbase

	hosts []types.Host
}

func newSimpleInMemCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaApi bool) *simpleInMemCluster {
	cluster := newCluster(clusterConfig, sourceAddr, addedViaApi, nil)

	return &simpleInMemCluster{
		dynamicClusterbase: dynamicClusterbase{
			cluster: cluster,
		},
	}
}

func (sc *simpleInMemCluster) UpdateHosts(newHosts []types.Host) {
	var curHosts []types.Host
	copy(curHosts, sc.hosts)

	changed, finalHosts, hostsAdded, hostsRemoved := sc.updateDynamicHostList(newHosts, curHosts)

	if changed {
		sc.hosts = finalHosts
		sc.prioritySet.GetOrCreateHostSet(0).UpdateHosts(sc.hosts,
			nil, nil, nil, hostsAdded, hostsRemoved)
	}
}
