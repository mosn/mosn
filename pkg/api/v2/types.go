package v2

import (
	"net"
	"time"
)

type Metadata struct {
}

const (
	MaxRequestsPerConn  uint64 = 10000
	ConnBufferLimitByte uint32 = 16 * 1024
)

type ClusterType string

const (
	STATIC_CLUSTER  ClusterType = "STATIC"
	SIMPLE_CLUSTER  ClusterType = "SIMPLE"
	DYNAMIC_CLUSTER ClusterType = "DYNAMIC"
)

type SubClusterType string

const (
	CONFREG_CLUSTER SubClusterType = "CONFREG"
	// also , ZooKeeper
)

type LbType string

const (
	LB_RANDOM     LbType = "LB_RANDOM"
	LB_ROUNDROBIN LbType = "LB_ROUNDROBIN"
)

type Cluster struct {
	Name              string
	ClusterType       ClusterType
	SubClustetType    SubClusterType
	LbType            LbType
	MaxRequestPerConn uint64

	ConnBufferLimitBytes uint32
	HealthCheck          HealthCheck
	Spec                 ClusterSpecInfo
}

type Host struct {
	Address  string
	Hostname string
	Weight   uint32
}

type ListenerConfig struct {
	Name                                  string
	Addr                                  net.Addr
	ListenerTag                           uint64
	ListenerScope                         string
	BindToPort                            bool
	PerConnBufferLimitBytes               uint32
	HandOffRestoredDestinationConnections bool

	// used in inherit case
	InheritListener *net.TCPListener
	Remain          bool

	// log
	LogPath    string
	LogLevel   uint8
	AccessLogs []AccessLog

	// only used in http2 case
	DisableConnIo bool
}

type AccessLog struct {
	Path   string
	Format string
	// todo: add log filters
}

type TcpRoute struct {
	Cluster          string
	SourceAddrs      []net.Addr
	DestinationAddrs []net.Addr
}

type TcpProxy struct {
	Routes     []*TcpRoute
	AccessLogs []*AccessLog
}

type RpcRoute struct {
	Name    string
	Service string
	Cluster string
}

type RpcProxy struct {
	Routes []*RpcRoute
}

type FaultInject struct {
	DelayPercent  uint32
	DelayDuration uint64
}

type Proxy struct {
	DownstreamProtocol  string
	UpstreamProtocol    string
	SupportDynamicRoute bool
	Routes              []*BasicServiceRoute
}

type BasicServiceRoute struct {
	Name          string
	Service       string
	Cluster       string
	GlobalTimeout time.Duration
	RetryPolicy   *RetryPolicy
}

type RetryPolicy struct {
	RetryOn      bool
	RetryTimeout time.Duration
	NumRetries   int
}

type HealthCheck struct {
	Timeout            time.Duration
	HealthyThreshold   uint32
	UnhealthyThreshold uint32
	Interval           time.Duration
	IntervalJitter     time.Duration
	CheckPath          string
	ServiceName        string
}

type HealthCheckFilter struct {
	PassThrough                 bool
	CacheTime                   time.Duration
	Endpoint                    string
	ClusterMinHealthyPercentage map[string]float32
}

// currently only one subscribe allowed
type ClusterSpecInfo struct {
	Subscribes []SubscribeSpec
}

type SubscribeSpec struct {
	ServiceName string
}

type ServiceRegistryInfo struct{
	ServiceAppInfo  ApplicationInfo
	ServicePubInfo  []PublishInfo
}

type ApplicationInfo struct {
	AntShareCloud bool
	DataCenter    string
	AppName       string
	Zone          string
}

type PublishInfo struct{
	Pub  PublishContent
}

type PublishContent struct{
	ServiceName string
	PubData string
}