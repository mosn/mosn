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

package v2

import (
	"net"
	"time"
)

// Metadata field can be used to provide additional information about the route.
// It can be used for configuration, stats, and logging.
// The metadata should go under the filter namespace that will need it.
type Metadata map[string]string

// Network Filter's Name
const (
	DEFAULT_NETWORK_FILTER      = "proxy"
	TCP_PROXY                   = "tcp_proxy"
	FAULT_INJECT_NETWORK_FILTER = "fault_inject"
	RPC_PROXY                   = "rpc_proxy"
	X_PROXY                     = "x_proxy"
)

// ClusterType
type ClusterType string

// Group of cluster type
const (
	STATIC_CLUSTER  ClusterType = "STATIC"
	SIMPLE_CLUSTER  ClusterType = "SIMPLE"
	DYNAMIC_CLUSTER ClusterType = "DYNAMIC"
	EDS_CLUSTER     ClusterType = "EDS"
)

// LbType
type LbType string

// Group of load balancer type
const (
	LB_RANDOM     LbType = "LB_RANDOM"
	LB_ROUNDROBIN LbType = "LB_ROUNDROBIN"
)

// Cluster class
type Cluster struct {
	Name                 string
	ClusterType          ClusterType
	LbType               LbType
	MaxRequestPerConn    uint32
	ConnBufferLimitBytes uint32
	CirBreThresholds     CircuitBreakers
	OutlierDetection     OutlierDetection
	HealthCheck          HealthCheck
	Spec                 ClusterSpecInfo
	LBSubSetConfig       LBSubsetConfig
	TLS                  TLSConfig
	Hosts                []Host
}

// CircuitBreakers class
type CircuitBreakers struct {
	Thresholds []Thresholds
}

// Thresholds of CircuitBreakers
type Thresholds struct {
	Priority           RoutingPriority
	MaxConnections     uint32
	MaxPendingRequests uint32
	MaxRequests        uint32
	MaxRetries         uint32
}

// OutlierDetection value for upstream
type OutlierDetection struct {
	Consecutive5xx                     uint32
	Interval                           time.Duration
	BaseEjectionTime                   time.Duration
	MaxEjectionPercent                 uint32
	ConsecutiveGatewayFailure          uint32
	EnforcingConsecutive5xx            uint32
	EnforcingConsecutiveGatewayFailure uint32
	EnforcingSuccessRate               uint32
	SuccessRateMinimumHosts            uint32
	SuccessRateRequestVolume           uint32
	SuccessRateStdevFactor             uint32
}

// RoutingPriority
type RoutingPriority string

// Group of routing priority
const (
	DEFAULT RoutingPriority = "DEFAULT"
	HIGH    RoutingPriority = "HIGH"
)

// Host with MetaData
type Host struct {
	Address  string
	Hostname string
	Weight   uint32
	MetaData Metadata
}

// ListenerConfig with FilterChains
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
	Inspector                             bool          // TLS inspector
}

// AccessLog with log path and log format
type AccessLog struct {
	Path   string
	Format string
	// todo: add log filters
}

// TLSConfig
type TLSConfig struct {
	Status       bool
	Type         string
	ServerName   string
	CACert       string
	CertChain    string
	PrivateKey   string
	VerifyClient bool
	InsecureSkip bool
	CipherSuites string
	EcdhCurves   string
	MinVersion   string
	MaxVersion   string
	ALPN         string
	Ticket       string
	ExtendVerify map[string]interface{}
}

// TCPRoute
type TCPRoute struct {
	Cluster          string
	SourceAddrs      []net.Addr
	DestinationAddrs []net.Addr
}

// TCPProxy
type TCPProxy struct {
	Routes []*TCPRoute
}

// RPCRoute
type RPCRoute struct {
	Name    string
	Service string
	Cluster string
}

// RPCProxy
type RPCProxy struct {
	Routes []*RPCRoute
}

// FaultInject
type FaultInject struct {
	DelayPercent  uint32
	DelayDuration uint64
}

type Proxy struct {
	Name                string
	DownstreamProtocol  string
	UpstreamProtocol    string
	SupportDynamicRoute bool
	BasicRoutes         []*BasicServiceRoute
	VirtualHosts        []*VirtualHost
	ValidateClusters    bool
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

// HealthCheck
type HealthCheck struct {
	Protocol           string
	ProtocolCode       byte // used by sofa rpc
	Timeout            time.Duration
	Interval           time.Duration
	IntervalJitter     time.Duration
	HealthyThreshold   uint32
	UnhealthyThreshold uint32
	CheckPath          string
	ServiceName        string
}

// HealthCheckFilter
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

// FilterChain wraps a set of match criteria, an option TLS context,
// a set of filters, and various other parameters.
type FilterChain struct {
	FilterChainMatch string
	TLS              TLSConfig
	Filters          []Filter
}

// Filter for network and stream
type Filter struct {
	Name   string
	Config map[string]interface{}
}

// VirtualHost
// An array of virtual hosts that make up the route table.
type VirtualHost struct {
	Name            string
	Domains         []string
	Routers         []Router
	RequireTLS      string
	VirtualClusters []VirtualCluster
}

// Router, the list of routes that will be matched, in order, for incoming requests.
// The first route that matches will be used.
type Router struct {
	Match     RouterMatch
	Route     RouteAction
	Redirect  RedirectAction
	Metadata  Metadata
	Decorator Decorator
}

// Decorator
type Decorator string

// RedirectAction
// Return a redirect.
type RedirectAction struct {
	HostRedirect string
	PathRedirect string
	ResponseCode uint32
}

// RouterMatch
// Route matching parameters
type RouterMatch struct {
	Prefix        string
	Path          string
	Regex         string
	CaseSensitive bool
	Runtime       RuntimeUInt32
	Headers       []HeaderMatcher
}

// RouteAction
// Route request to some upstream clusters.
type RouteAction struct {
	ClusterName        string
	ClusterHeader      string
	TotalClusterWeight uint32 // total weight of weighted clusters, such as 100
	WeightedClusters   []WeightedCluster
	MetadataMatch      Metadata
	Timeout            time.Duration
	RetryPolicy        *RetryPolicy
}

// WeightedCluster.
// Multiple upstream clusters unsupport stream filter type:  healthcheckcan be specified for a given route.
// The request is routed to one of the upstream
// clusters based on weights assigned to each cluster
type WeightedCluster struct {
	Cluster          ClusterWeight
	RuntimeKeyPrefix string // not used currently
}

// ClusterWeight.
// clusters along with weights that indicate the percentage
// of traffic to be forwarded to each cluste
type ClusterWeight struct {
	Name          string
	Weight        uint32
	MetadataMatch Metadata
}

// RuntimeUInt32
// Indicates that the route should additionally match on a runtime key
type RuntimeUInt32 struct {
	DefaultValue uint32
	RuntimeKey   string
}

// HeaderMatcher specifies a set of headers that the route should match on.
type HeaderMatcher struct {
	Name  string
	Value string
	Regex bool
}

// VirtualCluster is a way of specifying a regex matching rule against certain important endpoints
// such that statistics are generated explicitly for the matched requests
type VirtualCluster struct {
	Pattern string
	Name    string
	Method  string // http.Request.Method
}
