package types

import (
	"context"
	"github.com/rcrowley/go-metrics"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"

	"net"
)

type ClusterManager interface {
	AddOrUpdatePrimaryCluster(cluster v2.Cluster) bool

	SetInitializedCb(cb func())

	Clusters() map[string]Cluster

	Get(cluster string, context context.Context) ClusterSnapshot

	// temp interface todo: remove it
	UpdateClusterHosts(cluster string, priority uint32, hosts []v2.Host) error

	HttpConnPoolForCluster(cluster string, protocol Protocol, context context.Context) ConnectionPool

	XprotocolConnPoolForCluster(cluster string, protocol Protocol, context context.Context) ConnectionPool

	TcpConnForCluster(cluster string, context context.Context) CreateConnectionData

	SofaRpcConnPoolForCluster(cluster string, context context.Context) ConnectionPool

	RemovePrimaryCluster(cluster string) bool

	Shutdown() error

	SourceAddress() net.Addr

	VersionInfo() string

	LocalClusterName() string

	ClusterExist(clusterName string) bool
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
	Primary   InitializePhase = "Primary"
	Secondary InitializePhase = "Secondary"
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

	UpdateHosts(hosts []Host, healthyHost []Host, hostsPerLocality [][]Host,
		healthyHostPerLocality [][]Host, hostsAdded []Host, hostsRemoved []Host)

	Priority() uint32
}

type HealthFlag int

const (
	FAILED_ACTIVE_HC     HealthFlag = 0x1
	FAILED_OUTLIER_CHECK HealthFlag = 0x02
)

type Host interface {
	HostInfo

	CreateConnection(context context.Context) CreateConnectionData

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

	AddressString() string

	HostStats() HostStats

	// TODO: add deploy locality
}

type HostStats struct {
	Namespace                                      string
	UpstreamConnectionTotal                        metrics.Counter
	UpstreamConnectionClose                        metrics.Counter
	UpstreamConnectionActive                       metrics.Counter
	UpstreamConnectionTotalHttp1                   metrics.Counter
	UpstreamConnectionTotalHttp2                   metrics.Counter
	UpstreamConnectionTotalSofaRpc                 metrics.Counter
	UpstreamConnectionConFail                      metrics.Counter
	UpstreamConnectionLocalClose                   metrics.Counter
	UpstreamConnectionRemoteClose                  metrics.Counter
	UpstreamConnectionLocalCloseWithActiveRequest  metrics.Counter
	UpstreamConnectionRemoteCloseWithActiveRequest metrics.Counter
	UpstreamConnectionCloseNotify                  metrics.Counter
	UpstreamRequestTotal                           metrics.Counter
	UpstreamRequestActive                          metrics.Counter
	UpstreamRequestLocalReset                      metrics.Counter
	UpstreamRequestRemoteReset                     metrics.Counter
	UpstreamRequestTimeout                         metrics.Counter
	UpstreamRequestFailureEject                    metrics.Counter
	UpstreamRequestPendingOverflow                 metrics.Counter
}

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

	MaxRequestsPerConn() uint32

	Stats() ClusterStats

	ResourceManager() ResourceManager

        TLSMng() TLSContextManager
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

type ClusterStats struct {
	Namespace                                      string
	UpstreamConnectionTotal                        metrics.Counter
	UpstreamConnectionClose                        metrics.Counter
	UpstreamConnectionActive                       metrics.Counter
	UpstreamConnectionTotalHttp1                   metrics.Counter
	UpstreamConnectionTotalHttp2                   metrics.Counter
	UpstreamConnectionTotalSofaRpc                 metrics.Counter
	UpstreamConnectionConFail                      metrics.Counter
	UpstreamConnectionRetry                        metrics.Counter
	UpstreamConnectionLocalClose                   metrics.Counter
	UpstreamConnectionRemoteClose                  metrics.Counter
	UpstreamConnectionLocalCloseWithActiveRequest  metrics.Counter
	UpstreamConnectionRemoteCloseWithActiveRequest metrics.Counter
	UpstreamConnectionCloseNotify                  metrics.Counter
	UpstreamBytesRead                              metrics.Counter
	UpstreamBytesReadCurrent                       metrics.Gauge
	UpstreamBytesWrite                             metrics.Counter
	UpstreamBytesWriteCurrent                      metrics.Gauge
	UpstreamRequestTotal                           metrics.Counter
	UpstreamRequestActive                          metrics.Counter
	UpstreamRequestLocalReset                      metrics.Counter
	UpstreamRequestRemoteReset                     metrics.Counter
	UpstreamRequestRetry                           metrics.Counter
	UpstreamRequestTimeout                         metrics.Counter
	UpstreamRequestFailureEject                    metrics.Counter
	UpstreamRequestPendingOverflow                 metrics.Counter
}

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

type ClusterConfigFactoryCb interface {
	UpdateClusterConfig(configs []v2.Cluster) error
}

type ClusterHostFactoryCb interface {
	UpdateClusterHost(cluster string, priority uint32, hosts []v2.Host) error
}

type ClusterManagerFilter interface {
	OnCreated(cccb ClusterConfigFactoryCb, chcb ClusterHostFactoryCb)
}

type RegisterUpstreamUpdateMethodCb interface {
	TriggerClusterUpdate(clusterName string, hosts []v2.Host)
	GetClusterNameByServiceName(serviceName string) string
}

// SubSetLoadBalancer
type SubSetLoadBalancer interface {
	// Find a host from the subsets, context is LoadBalancerContext
	TryChooseHostFromContext(context *context.Context, hostChosen *bool)
	
	HostMatchesDefaultSubset(host Host) bool
	
	HostMatches(kvs *SubsetMetadata, host *Host) bool
	
	// Iterates over the given metadata match criteria (which must be lexically sorted by key) and find
	// a matching LbSubsetEnryPtr, if any.
	FindSubset(matches []MetadataMatchEntry)*LBSubsetEntry
	
	// Given a vector of key-values (from extractSubsetMetadata), recursively finds the matching
	// LbSubsetEntryPtr.
	FindOrCreateSubset(subsets *LbSubsetMap, kvs *SubsetMetadata, idx uint32)
	
	// Iterates over subset_keys looking up values from the given host's metadata. Each key-value pair
	// is appended to kvs. Returns a non-empty value if the host has a value for each key.
	ExtractSubsetMetadata(subsetKeys []string, host Host)
	
	//extractSubsetMetadata2(subsetKeys []string, host Host)
}

type FallBackPolicy uint8

const (
	NoFallBack FallBackPolicy = 0
	AnyEndPoint FallBackPolicy = 1
	DefaultSubsetDefaultSubset FallBackPolicy = 2
)

// Forms a trie-like structure. Route Metadata requires lexically sorted
type LbSubsetMap struct {
	lbSubsetMap  map[string]ValueSubsetMap
}

//
type ValueSubsetMap struct {
	valueSubsetMap    map[interface{}]*LBSubsetEntry
}

type LBSubsetEntry struct {
	children       LbSubsetMap
	prioritySubset HostSet
}

type SubsetMetadata struct {
	subsetMatadata  []Pair
}

type Pair struct {
	T1   string
	T2   interface {}
}

type ResourcePriority uint8

const (
	Default ResourcePriority = 0
	High ResourcePriority = 1
)
