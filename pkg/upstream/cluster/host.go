package cluster

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/rcrowley/go-metrics"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type hostSet struct {
	priority                uint32
	hosts                   []types.Host
	healthyHosts            []types.Host
	hostsPerLocality        [][]types.Host
	healthyHostsPerLocality [][]types.Host
	mux                     sync.RWMutex
	updateCallbacks         []types.MemberUpdateCallback
}

func (hs *hostSet) Hosts() []types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.hosts
}

func (hs *hostSet) HealthyHosts() []types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	// TODO: support health check
	return hs.hosts
}

func (hs *hostSet) HostsPerLocality() [][]types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.hostsPerLocality
}

func (hs *hostSet) HealthHostsPerLocality() [][]types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.healthyHostsPerLocality
}

func (hs *hostSet) UpdateHosts(hosts []types.Host, healthyHosts []types.Host, hostsPerLocality [][]types.Host,
	healthyHostsPerLocality [][]types.Host, hostsAdded []types.Host, hostsRemoved []types.Host) {
	hs.mux.Lock()
	defer hs.mux.Unlock()

	hs.hosts = hosts
	hs.healthyHosts = healthyHosts
	hs.hostsPerLocality = hostsPerLocality
	hs.healthyHostsPerLocality = healthyHostsPerLocality

	for _, updateCb := range hs.updateCallbacks {
		updateCb(hs.priority, hostsAdded, hostsRemoved)
	}
}

func (hs *hostSet) Priority() uint32 {
	return hs.priority
}

func (hs *hostSet) addMemberUpdateCb(cb types.MemberUpdateCallback) {
	hs.updateCallbacks = append(hs.updateCallbacks, cb)
}

// Host
type host struct {
	hostInfo
	weight uint32
	used   bool

	healthFlags uint64
}

func newHost(config v2.Host, clusterInfo types.ClusterInfo) types.Host {
	addr, _ := net.ResolveTCPAddr("tcp", config.Address)

	return &host{
		hostInfo: newHostInfo(addr, config, clusterInfo),
		weight:   config.Weight,
	}
}

func newHostStats(config v2.Host) types.HostStats {
	nameSpace := fmt.Sprintf("host.%s", config.Address)

	return types.HostStats{
		Namespace:                                      nameSpace,
		UpstreamConnectionTotal:                        metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_total"), nil),
		UpstreamConnectionClose:                        metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_close"), nil),
		UpstreamConnectionActive:                       metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_active"), nil),
		UpstreamConnectionTotalHttp1:                   metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_total_http1"), nil),
		UpstreamConnectionTotalHttp2:                   metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_total_http2"), nil),
		UpstreamConnectionTotalSofaRpc:                 metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_total_sofarpc"), nil),
		UpstreamConnectionConFail:                      metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_con_fail"), nil),
		UpstreamConnectionLocalClose:                   metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_local_close"), nil),
		UpstreamConnectionRemoteClose:                  metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_remote_close"), nil),
		UpstreamConnectionLocalCloseWithActiveRequest:  metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_local_close_with_active_request"), nil),
		UpstreamConnectionRemoteCloseWithActiveRequest: metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_remote_close_with_active_request"), nil),
		UpstreamConnectionCloseNotify:                  metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_connection_close_notify"), nil),
		UpstreamRequestTotal:                           metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_request_total"), nil),
		UpstreamRequestActive:                          metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_request_active"), nil),
		UpstreamRequestLocalReset:                      metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_request_local_reset"), nil),
		UpstreamRequestRemoteReset:                     metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_request_remote_reset"), nil),
		UpstreamRequestTimeout:                         metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_request_timeout"), nil),
		UpstreamRequestFailureEject:                    metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_failure_eject"), nil),
		UpstreamRequestPendingOverflow:                 metrics.GetOrRegisterCounter(fmt.Sprintf("%s.%s", nameSpace, "upstream_request_pending_overflow"), nil),
	}
}

func (h *host) CreateConnection(context context.Context) types.CreateConnectionData {
	logger := log.ByContext(context)
	clientConn := network.NewClientConnection(h.clusterInfo.SourceAddress(), h.address, nil, logger)
	clientConn.SetBufferLimit(h.clusterInfo.ConnBufferLimitBytes())

	return types.CreateConnectionData{
		Connection: clientConn,
		HostInfo:   &h.hostInfo,
	}
}

func (h *host) Counters() types.HostStats {
	return types.HostStats{}
}

func (h *host) Gauges() types.HostStats {
	return types.HostStats{}
}

func (h *host) ClearHealthFlag(flag types.HealthFlag) {
	h.healthFlags &= ^uint64(flag)
}

func (h *host) ContainHealthFlag(flag types.HealthFlag) bool {
	return h.healthFlags&uint64(flag) > 0
}

func (h *host) SetHealthFlag(flag types.HealthFlag) {
	h.healthFlags |= uint64(flag)
}

func (h *host) Health() bool {
	return true
}

func (h *host) SetHealthChecker(healthCheck types.HealthCheckHostMonitor) {
}

func (h *host) SetOutlierDetector(outlierDetector types.DetectorHostMonitor) {
}

func (h *host) Weight() uint32 {
	return h.weight
}

func (h *host) SetWeight(weight uint32) {
	h.weight = weight
}

func (h *host) Used() bool {
	return h.used
}

func (h *host) SetUsed(used bool) {
	h.used = used
}

// HostInfo
type hostInfo struct {
	hostname      string
	address       net.Addr
	addressString string
	canary        bool
	clusterInfo   types.ClusterInfo
	stats         types.HostStats

	// TODO: metadata, locality, outlier, healthchecker
}

func newHostInfo(addr net.Addr, config v2.Host, clusterInfo types.ClusterInfo) hostInfo {
	return hostInfo{
		address:       addr,
		addressString: config.Address,
		hostname:      config.Hostname,
		clusterInfo:   clusterInfo,
		stats:         newHostStats(config),
	}
}

func (hi *hostInfo) Hostname() string {
	return hi.hostname
}

func (hi *hostInfo) Canary() bool {
	return hi.canary
}

func (hi *hostInfo) Metadata() v2.Metadata {
	return v2.Metadata{}
}

func (hi *hostInfo) ClusterInfo() types.ClusterInfo {
	return hi.clusterInfo
}

func (hi *hostInfo) OutlierDetector() types.DetectorHostMonitor {
	return nil
}

func (hi *hostInfo) HealthChecker() types.HealthCheckHostMonitor {
	return nil
}

func (hi *hostInfo) Address() net.Addr {
	return hi.address
}

func (hi *hostInfo) AddressString() string {
	return hi.addressString
}

func (hi *hostInfo) HostStats() types.HostStats {
	return hi.stats
}
