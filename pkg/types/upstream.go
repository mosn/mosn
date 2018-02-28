package types

import (
	"net"
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg"
)

type ClusterManager interface {
	AddOrUpdatePrimaryCluster(cluster v2.Cluster) bool

	SetInitializedCb(cb func())

	Clusters() map[string]Cluster

	Get(cluster string, context context.Context) ClusterSnapshot

	HttpConnPoolForCluster(cluster string, priority pkg.Priority, protocol string, context context.Context) HttpConnectionPool

	TcpConnForCluster(cluster string, context context.Context) CreateConnectionData

	RemovePrimaryCluster(cluster string)

	Shutdown() error

	SourceAddress() net.Addr

	VersionInfo() string

	LocalClusterName() string
}

// thread-safe cluster snapshot
type ClusterSnapshot interface {
	PrioritySet() PrioritySet

	ClusterInfo() ClusterInfo

	LoadBalancer() LoadBalancer
}

type Cluster interface {
	Initialize(cb func())

	Info() ClusterInfo

	InitializePhase() InitializePhase

	PrioritySet() PrioritySet

	HealthChecker() HealthChecker

	OutlierDetector() Detector
}

type InitializePhase string

const (
	Primary   InitializePhase = "Primary";
	Secondary InitializePhase = "Secondary";
)

type MemberUpdateCallback func(priority uint32, hostsAdded []Host, hostsRemoved []Host)

type PrioritySet interface {
	GetOrCreateHostSet(priority uint32) HostSet

	AddMemberUpdateCb(cb MemberUpdateCallback)

	HostSetsByPriority() []HostSet
}

type HostSet interface {
	Hosts() []Host

	HealthyHosts() []Host

	HostsPerLocality() [][]Host

	HealthHostsPerLocality() [][]Host

	UpdateHosts(hosts []Host, healthyHost []Host, hostsPerLocality []Host,
		healthyHostPerLocality []Host, hostsAdded []Host, hostsRemoved []Host)

	Priority() uint32
}

type HealthFlag int

const (
	FAILED_ACTIVE_HC     HealthFlag = 0x1
	FAILED_OUTLIER_CHECK HealthFlag = 0x02
)

type Host interface {
	HostInfo

	CreateConnection() CreateConnectionData

	Counters() HostStats

	Gauges() HostStats

	ClearHealthFlag(flag HealthFlag)

	ContainHealthFlag(flag HealthFlag) bool

	SetHealthFlag(flag HealthFlag)

	Health() bool

	SetHealthChecker(healthCheck HealthCheckHostMonitor)

	SetOutlierDetector(outlierDetector DetectorHostMonitor)

	Weight() uint32

	SetWeight(weight uint32)

	Used() bool

	SetUsed(used bool)
}

type HostInfo interface {
	Hostname() string

	Canary() bool

	Metadata() v2.Metadata

	ClusterInfo() ClusterInfo

	OutlierDetector() DetectorHostMonitor

	HealthChecker() HealthCheckHostMonitor

	Address() net.Addr

	HostStats() HostStats

	// TODO: add deploy locality
}

type HostStats []pkg.Stat

var (
	cxTotal = pkg.Stat{
		Key:  "cx_total",
		Type: pkg.COUNTER,
	}

	cxActive = pkg.Stat{
		Key:  "cx_active",
		Type: pkg.GAUGE,
	}

	// TODO
)

type ClusterInfo interface {
	Name() string

	LbType() LoadBalancerType

	AddedViaApi() bool

	SourceAddress() net.Addr

	ConnectTimeout() int

	ConnBufferLimitBytes() uint32

	Features() int

	Metadata() v2.Metadata

	DiscoverType() string

	MaintenanceMode() bool

	MaxRequestsPerConn() uint64

	Stats() ClusterStats

	ResourceManager() ResourceManager
}

type ResourceManager interface {
	ConnectionResource() Resource

	PendingRequests() Resource

	Requests() Resource
}

type Resource interface {
	CanCreate() bool
	Increase()
	Decrease()
	Max() uint64
}

type ClusterStats []pkg.Stat

var (
	lbHealthyPanic = pkg.Stat{
		Key:  "lb_healthy_panic",
		Type: pkg.COUNTER,
	}

	lbLocalClusterNotOk = pkg.Stat{
		Key:  "lb_local_cluster_not_ok",
		Type: pkg.COUNTER,
	}

	// TODO
)

type CreateConnectionData struct {
	Connection ClientConnection
	HostInfo   HostInfo
}

// a simple in mem cluster
type SimpleCluster interface {
	UpdateHosts(newHosts []Host)
}

type LoadBalancerType string

const (
	RoundRobin LoadBalancerType = "RoundRobin"
	Random     LoadBalancerType = "Random"
)

type LoadBalancer interface {
	ChooseHost(context context.Context) Host
}
