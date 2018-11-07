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

	"istio.io/api/mixer/v1/config/client"
)

// Metadata field can be used to provide additional information about the route.
// It can be used for configuration, stats, and logging.
// The metadata should go under the filter namespace that will need it.
type Metadata map[string]string

// Network Filter's Type
const (
	CONNECTION_MANAGER          = "connection_manager"
	DEFAULT_NETWORK_FILTER      = "proxy"
	TCP_PROXY                   = "tcp_proxy"
	FAULT_INJECT_NETWORK_FILTER = "fault_inject"
	RPC_PROXY                   = "rpc_proxy"
	X_PROXY                     = "x_proxy"
	MIXER                       = "mixer"
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

// RoutingPriority
type RoutingPriority string

// Group of routing priority
const (
	DEFAULT RoutingPriority = "DEFAULT"
	HIGH    RoutingPriority = "HIGH"
)

// Cluster represents a cluster's information
type Cluster struct {
	Name                 string           `json:"name"`
	ClusterType          ClusterType      `json:"type"`
	SubType              string           `json:"sub_type"` //not used yet
	LbType               LbType           `json:"lb_type"`
	MaxRequestPerConn    uint32           `json:"max_request_per_conn"`
	ConnBufferLimitBytes uint32           `json:"conn_buffer_limit_bytes"`
	CirBreThresholds     CircuitBreakers  `json:"circuit_breakers,omitempty"`
	OutlierDetection     OutlierDetection `json:"outlier_detection,omitempty"` //not used yet
	HealthCheck          HealthCheck      `json:"health_check,omitempty"`
	Spec                 ClusterSpecInfo  `json:"spec,omitempty"`
	LBSubSetConfig       LBSubsetConfig   `json:"lb_subset_config,omitempty"`
	TLS                  TLSConfig        `json:"tls_context,omitempty"`
	Hosts                []Host           `json:"hosts"`
}

// HealthCheck is a configuration of health check
// use DurationConfig to parse string to time.Duration
type HealthCheck struct {
	HealthCheckConfig
	ProtocolCode   byte          `json:"-"`
	Timeout        time.Duration `json:"-"`
	Interval       time.Duration `json:"-"`
	IntervalJitter time.Duration `json:"-"`
}

// Host represenets a host information
type Host struct {
	HostConfig
	MetaData Metadata `json:"-"`
}

// Listener contains the listener's information
type Listener struct {
	ListenerConfig
	Addr                    net.Addr         `json:"-"`
	ListenerTag             uint64           `json:"-"`
	ListenerScope           string           `json:"-"`
	PerConnBufferLimitBytes uint32           `json:"-"` // do not support config
	InheritListener         *net.TCPListener `json:"-"`
	Remain                  bool             `json:"-"`
	LogLevel                uint8            `json:"-"`
	DisableConnIo           bool             `json:"-"`
}

// TCPRoute
type TCPRoute struct {
	Cluster          string
	SourceAddrs      []CidrRange
	DestinationAddrs []CidrRange
	SourcePort       string
	DestinationPort  string
}

// CidrRange
type CidrRange struct {
	Address string
	Length  uint32
	IpNet   *net.IPNet
}

// HealthCheckFilter
type HealthCheckFilter struct {
	HealthCheckFilterConfig
	CacheTime time.Duration `json:"-"`
}

// FaultInject
type FaultInject struct {
	FaultInjectConfig
	DelayDuration uint64 `json:"-"`
}

type Mixer struct {
	client.HttpClientConfig
}

// Router, the list of routes that will be matched, in order, for incoming requests.
// The first route that matches will be used.
type Router struct {
	RouterConfig
	Metadata Metadata `json:"-"`
}

// RouteAction represents the information of route request to upstream clusters
type RouteAction struct {
	RouterActionConfig
	MetadataMatch Metadata      `json:"-"`
	Timeout       time.Duration `json:"-"`
}

// Decorator
type Decorator string

// ClusterWeight.
// clusters along with weights that indicate the percentage
// of traffic to be forwarded to each cluster
type ClusterWeight struct {
	ClusterWeightConfig
	MetadataMatch Metadata `json:"-"`
}

// RetryPolicy represents the retry parameters
type RetryPolicy struct {
	RetryPolicyConfig
	RetryTimeout time.Duration `json:"-"`
}

// CircuitBreakers is a configuration of circuit breakers
// CircuitBreakers implements json.Marshaler and json.Unmarshaler
type CircuitBreakers struct {
	Thresholds []Thresholds
}

type Thresholds struct {
	Priority           RoutingPriority `json:"priority"`
	MaxConnections     uint32          `json:"max_connections"`
	MaxPendingRequests uint32          `json:"max_pending_requests"`
	MaxRequests        uint32          `json:"max_requests"`
	MaxRetries         uint32          `json:"max_retries"`
}

// OutlierDetection not used yet
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

// ClusterSpecInfo is a configuration of subscribe
type ClusterSpecInfo struct {
	Subscribes []SubscribeSpec `json:"subscribe,omitempty"`
}

// SubscribeSpec describes the subscribe server
type SubscribeSpec struct {
	ServiceName string `json:"service_name,omitempty"`
}

// LBSubsetConfig is a configuration of load balance subset
type LBSubsetConfig struct {
	FallBackPolicy  uint8             `json:"fall_back_policy"`
	DefaultSubset   map[string]string `json:"default_subset"`
	SubsetSelectors [][]string        `json:"subset_selectors"`
}

// TLSConfig is a configuration of tls context
type TLSConfig struct {
	Status       bool                   `json:"status"`
	Type         string                 `json:"type"`
	ServerName   string                 `json:"server_name,omitempty"`
	CACert       string                 `json:"ca_cert,omitempty"`
	CertChain    string                 `json:"cert_chain,omitempty"`
	PrivateKey   string                 `json:"private_key,omitempty"`
	VerifyClient bool                   `json:"verify_client,omitempty"`
	InsecureSkip bool                   `json:"insecure_skip,omitempty"`
	CipherSuites string                 `json:"cipher_suites,omitempty"`
	EcdhCurves   string                 `json:"ecdh_curves,omitempty"`
	MinVersion   string                 `json:"min_version,omitempty"`
	MaxVersion   string                 `json:"max_version,omitempty"`
	ALPN         string                 `json:"alpn,omitempty"`
	Ticket       string                 `json:"ticket,omitempty"`
	Fallback     bool                   `json:"fall_back, omitempty"`
	ExtendVerify map[string]interface{} `json:"extend_verify,omitempty"`
}

// AccessLog for making up access log
type AccessLog struct {
	Path   string `json:"log_path,omitempty"`
	Format string `json:"log_format,omitempty"`
}

// FilterChain wraps a set of match criteria, an option TLS context,
// a set of filters, and various other parameters.
type FilterChain struct {
	FilterChainMatch string    `json:"match,omitempty"`
	TLS              TLSConfig `json:"tls_context,omitempty"`
	Filters          []Filter  `json:"filters"` // "proxy" and "connection_manager" used at this time
}

// Filter is a config to make up a filter
type Filter struct {
	Type   string                 `json:"type,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// Implements of filter config

// TCPProxy
type TCPProxy struct {
	StatPrefix         string         `json:"stat_prefix,omitempty"`
	Cluster            string         `json:"cluster,omitempty"`
	IdleTimeout        *time.Duration `json:"idle_timeout,omitempty"`
	MaxConnectAttempts uint32         `json:"max_connect_attempts,omitempty"`
	Routes             []*TCPRoute    `json:"routes,omitempty"`
}

// WebSocketProxy
type WebSocketProxy struct {
	StatPrefix         string
	IdleTimeout        *time.Duration
	MaxConnectAttempts uint32
}

// Proxy
type Proxy struct {
	Name               string                 `json:"name"`
	DownstreamProtocol string                 `json:"downstream_protocol"`
	UpstreamProtocol   string                 `json:"upstream_protocol"`
	RouterConfigName   string                 `json:"router_config_name"`
	ValidateClusters   bool                   `json:"validate_clusters"`
	ExtendConfig       map[string]interface{} `json:"extend_config"`
}

// HeaderValueOption is header name/value pair plus option to control append behavior.
type HeaderValueOption struct {
	Header *HeaderValue `json:"header"`
	Append *bool        `json:"append"`
}

// HeaderValue is header name/value pair.
type HeaderValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RouterConfiguration is a filter for routers
// Filter type is:  "CONNECTION_MANAGER"
type RouterConfiguration struct {
	RouterConfigName        string               `json:"router_config_name"`
	VirtualHosts            []*VirtualHost       `json:"virtual_hosts"`
	RequestHeadersToAdd     []*HeaderValueOption `json:"request_headers_to_add"`
	ResponseHeadersToAdd    []*HeaderValueOption `json:"response_headers_to_add"`
	ResponseHeadersToRemove []string             `json:"response_headers_to_remove"`
}

// VirtualHost is used to make up the route table
type VirtualHost struct {
	Name                    string               `json:"name"`
	Domains                 []string             `json:"domains"`
	VirtualClusters         []VirtualCluster     `json:"virtual_clusters"`
	Routers                 []Router             `json:"routers"`
	RequireTLS              string               `json:"require_tls"` // not used yet
	RequestHeadersToAdd     []*HeaderValueOption `json:"request_headers_to_add"`
	ResponseHeadersToAdd    []*HeaderValueOption `json:"response_headers_to_add"`
	ResponseHeadersToRemove []string             `json:"response_headers_to_remove"`
}

// VirtualCluster is a way of specifying a regex matching rule against certain important endpoints
// such that statistics are generated explicitly for the matched requests
type VirtualCluster struct {
	Pattern string `json:"pattern"`
	Name    string `json:"name"`
	Method  string `json:"method"`
}

// RouterMatch represents the route matching parameters
type RouterMatch struct {
	Prefix        string          `json:"prefix"` // Match request's Path with Prefix Comparing
	Path          string          `json:"path"`   // Match request's Path with Exact Comparing
	Regex         string          `json:"regex"`  // Match request's Path with Regex Comparing
	CaseSensitive bool            `json:"case_sensitive"`
	Runtime       RuntimeUInt32   `json:"runtime"`
	Headers       []HeaderMatcher `json:"headers"` // Match request's Headers
}

// RedirectAction represents the redirect parameters
type RedirectAction struct {
	HostRedirect string `json:"host_redirect"`
	PathRedirect string `json:"path_redirect"`
	ResponseCode uint32 `json:"response_code"`
}

// WeightedCluster.
// Multiple upstream clusters unsupport stream filter type:  healthcheckcan be specified for a given route.
// The request is routed to one of the upstream
// clusters based on weights assigned to each cluster
type WeightedCluster struct {
	Cluster          ClusterWeight `json:"cluster"`
	RuntimeKeyPrefix string        `json:"runtime_key_prefix"` // not used currently
}

// RuntimeUInt32 indicates that the route should additionally match on a runtime key
type RuntimeUInt32 struct {
	DefaultValue uint32 `json:"default_value"`
	RuntimeKey   string `json:"runtime_key"`
}

// HeaderMatcher specifies a set of headers that the route should match on.
type HeaderMatcher struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Regex bool   `json:"regex"`
}

// XProxyExtendConfig
type XProxyExtendConfig struct {
	SubProtocol string `json:"sub_protocol"`
}

// ServiceRegistryInfo
type ServiceRegistryInfo struct {
	ServiceAppInfo ApplicationInfo `json:"application"`
	ServicePubInfo []PublishInfo   `json:"publish_info,omitempty"`
}
type ApplicationInfo struct {
	AntShareCloud bool   `json:"ant_share_cloud"`
	DataCenter    string `json:"data_center,omitempty"`
	AppName       string `json:"app_name,omitempty"`
	Zone          string `json:"zone"`
}

// PublishInfo implements json.Marshaler and json.Unmarshaler
type PublishInfo struct {
	Pub PublishContent
}

type PublishContent struct {
	ServiceName string `json:"service_name,omitempty"`
	PubData     string `json:"pub_data,omitempty"`
}
