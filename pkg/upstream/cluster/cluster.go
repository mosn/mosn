package cluster

import (
	"fmt"
	"github.com/rcrowley/go-metrics"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"net"
	"sync"
)

// Cluster
type cluster struct {
	initializationStarted          bool
	initializationCompleteCallback func()
	prioritySet                    *prioritySet
	info                           *clusterInfo
	mux                            sync.RWMutex
	initHelper                     concreteClusterInitHelper
}

type concreteClusterInitHelper interface {
	Init()
}

func NewCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaApi bool) types.Cluster {
	var newCluster types.Cluster

	switch clusterConfig.ClusterType {
	//todo: add individual cluster for confreg
	case v2.SIMPLE_CLUSTER, v2.DYNAMIC_CLUSTER:
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
	mux             sync.RWMutex
}

func (ps *prioritySet) GetOrCreateHostSet(priority uint32) types.HostSet {
	ps.mux.Lock()
	defer ps.mux.Unlock()

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
	ps.mux.RLock()
	defer ps.mux.RUnlock()

	return ps.hostSets
}
