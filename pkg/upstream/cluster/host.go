package cluster

import (
	"net"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
)

type hostSet struct {
	priority                uint32
	hosts                   []types.Host
	healthyHosts            []types.Host
	hostsPerLocality        []types.Host
	healthyHostsPerLocality []types.Host
	updateCallbacks         []types.MemberUpdateCallback
}

func (hs *hostSet) Hosts() []types.Host {
	return hs.hosts
}

func (hs *hostSet) HealthyHosts() []types.Host {
	// TODO: support health check
	return hs.hosts
}

func (hs *hostSet) HostsPerLocality() [][]types.Host {
	return nil
}

func (hs *hostSet) HealthHostsPerLocality() [][]types.Host {
	return nil
}

func (hs *hostSet) UpdateHosts(hosts []types.Host, healthyHosts []types.Host, hostsPerLocality []types.Host,
	healthyHostsPerLocality []types.Host, hostsAdded []types.Host, hostsRemoved []types.Host) {
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
		hostInfo: hostInfo{
			address:     addr,
			hostname:    config.Hostname,
			clusterInfo: clusterInfo,
		},
		weight: config.Weight,
	}
}

func (h *host) CreateConnection() types.CreateConnectionData {
	clientConn := network.NewClientConnection(h.clusterInfo.SourceAddress(), h.address, nil)
	clientConn.SetBufferLimit(h.clusterInfo.ConnBufferLimitBytes())

	return types.CreateConnectionData{
		Connection: clientConn,
		HostInfo:   &h.hostInfo,
	}
}

func (h *host) Counters() types.HostStats {
	return nil
}

func (h *host) Gauges() types.HostStats {
	return nil
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
}

func (h *host) Used() bool {
	return h.used
}

func (h *host) SetUsed(used bool) {
	h.used = used
}

// HostInfo
type hostInfo struct {
	hostname    string
	address     net.Addr
	canary      bool
	clusterInfo types.ClusterInfo
	// TODO: metadata, locality, stats, outlier, healthchecker
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

func (hi *hostInfo) HostStats() types.HostStats {
	return nil
}
