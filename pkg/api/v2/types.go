package v2

import (
	"net"
	"time"
)

type Metadata map[string]string

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
	Name                 string
	ClusterType          ClusterType
	SubClustetType       SubClusterType
	ConnBufferLimitBytes uint32
	LbType               LbType
	MaxRequestPerConn    uint32
	CirBreThresholds     CircuitBreakers
	HealthCheck          HealthCheck
	Spec                 ClusterSpecInfo
	LBSubSetConfig       LBSubsetConfig
	TLS               TLSConfig
}

type CircuitBreakers struct {
	Thresholds []Thresholds
}

type Thresholds struct {
	Priority           RoutingPriority
	MaxConnections     uint32
	MaxPendingRequests uint32
	MaxRequests        uint32
	MaxRetries         uint32
}

type RoutingPriority string

const (
	DEFAULT RoutingPriority = "DEFAULT"
	HIGH    RoutingPriority = "HIGH"
)

type Host struct {
	Address  string
	Hostname string
	Weight   uint32
	MetaData Metadata
}

type ListenerConfig struct {
	Name                                  string
	Addr                                  net.Addr
	ListenerTag                           uint64
	ListenerScope                         string
	BindToPort                            bool
	PerConnBufferLimitBytes               uint32
	HandOffRestoredDestinationConnections bool
	InheritListener                       *net.TCPListener // used in inherit case
	Remain                                bool
	LogPath                               string // log
	LogLevel                              uint8
	AccessLogs                            []AccessLog
	DisableConnIo                         bool          // only used in http2 case
	FilterChains                          []FilterChain // FilterChains
}

type AccessLog struct {
	Path   string
	Format string
	// todo: add log filters
}

type TLSConfig struct {
	Status       bool
	ServerName   string
	CACert       string
	CertChain    string
	PrivateKey   string
	VerifyClient bool
	VerifyServer bool
	CipherSuites string
	EcdhCurves   string
	MinVersion   string
	MaxVersion   string
	ALPN         string
	Ticket       string
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
	BasicRoutes         []*BasicServiceRoute
	VirtualHosts        []*VirtualHost
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
	NumRetries   uint32
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

type ServiceRegistryInfo struct {
	ServiceAppInfo ApplicationInfo
	ServicePubInfo []PublishInfo
}

type ApplicationInfo struct {
	AntShareCloud bool
	DataCenter    string
	AppName       string
	Zone          string
}

type PublishInfo struct {
	Pub PublishContent
}

type PublishContent struct {
	ServiceName string
	PubData     string
}

type LBSubsetConfig struct {
	FallBackPolicy  uint8             // NoFallBack,...
	DefaultSubset   map[string]string // {e1,e2,e3}
	SubsetSelectors [][]string        // {{keys,},}, used to create subsets of hosts, pre-computing, sorted
}

type FilterChain struct {
	FilterChainMatch string
	TLS              TLSConfig
	Filters          []Filter
}

type Filter struct {
	Name   string
	Config map[string]interface{}
}

type VirtualHost struct {
	Name            string
	Domains         []string
	Routers         []Router
	RequireTls      string
	VirtualClusters []VirtualCluster
}

type Router struct {
	Match     RouterMatch
	Route     RouteAction
	Redirect  RedirectAction
	Metadata  Metadata
	Decorator Decorator
}

type Decorator string

type RedirectAction struct {
	HostRedirect string
	PathRedirect string
	ResponseCode uint32
}

type RouterMatch struct {
	Prefix        string
	Path          string
	Regex         string
	CaseSensitive bool
	Runtime       RuntimeUInt32
	Headers       []HeaderMatcher
}

type RouteAction struct {
	ClusterName      string
	ClusterHeader    string // used for http only
	WeightedClusters []WeightedCluster
	MetadataMatch    Metadata
	Timeout          time.Duration
	RetryPolicy      *RetryPolicy
}

type WeightedCluster struct {
	Clusters         ClusterWeight
	RuntimeKeyPrefix string // not used currently
}

type ClusterWeight struct {
	Name          string
	Weight        uint32
	MetadataMatch Metadata
}

type RuntimeUInt32 struct {
	DefaultValue uint32
	RuntimeKey   string
}

type HeaderMatcher struct {
	Name  string
	Value string
	Regex bool
}

type VirtualCluster struct {
	Pattern string
	Name    string
	Method  string // http.Request.Method
}
