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

package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/json-iterator/go"
)

//global instance for load & dump
var configPath string
var config MOSNConfig

type Metadata map[string]interface{}

// FilterChain wraps a set of match criteria, an option TLS context,
// a set of filters, and various other parameters.
type FilterChain struct {
	FilterChainMatch string         `json:"match,omitempty"`
	TLS              TLSConfig      `json:"tls_context,omitempty"`
	Filters          []FilterConfig `json:"filters"`
}

type Proxy struct {
	Name                string                  `json:"name"`
	DownstreamProtocol  string                  `json:"downstream_protocol"`
	UpstreamProtocol    string                  `json:"upstream_protocol"`
	SupportDynamicRoute bool                    `json:"support_dynamic_route"`
	BasicRoutes         []*v2.BasicServiceRoute `json:"basic_routes"` //not used anymore. todo: delete related logic
	VirtualHosts        []*VirtualHost          `json:"virtual_hosts"`
	ValidateClusters    bool                    `json:"validate_clusters"`
}

// VirtualHost
// An array of virtual hosts that make up the route table.
type VirtualHost struct {
	Name            string           `json:"name"`
	Domains         []string         `json:"domains"`
	Routers         []Router         `json:"routers"`
	RequireTLS      string           `json:"require_tls"`
	VirtualClusters []VirtualCluster `json:"virtual_clusters"`
}

// VirtualCluster is a way of specifying a regex matching rule against certain important endpoints
// such that statistics are generated explicitly for the matched requests
type VirtualCluster struct {
	Pattern string `json:"pattern"`
	Name    string `json:"name"`
	Method  string `json:"method"`
}

// Router, the list of routes that will be matched, in order, for incoming requests.
// The first route that matches will be used.
type Router struct {
	Match     RouterMatch    `json:"match"`
	Route     RouteAction    `json:"route"`
	Redirect  RedirectAction `json:"redirect"`
	Metadata  Metadata       `json:"metadata"`
	Decorator Decorator      `json:"decorator"`
}

// Decorator
type Decorator string

// RedirectAction
// Return a redirect.
type RedirectAction struct {
	HostRedirect string `json:"host_redirect"`
	PathRedirect string `json:"path_redirect"`
	ResponseCode uint32 `json:"response_code"`
}

// RouterMatch
// Route matching parameters
type RouterMatch struct {
	Prefix        string          `json:"prefix"`
	Path          string          `json:"path"`
	Regex         string          `json:"regex"`
	CaseSensitive bool            `json:"case_sensitive"`
	Runtime       RuntimeUInt32   `json:"runtime"`
	Headers       []HeaderMatcher `json:"headers"`
}

// HeaderMatcher specifies a set of headers that the route should match on.
type HeaderMatcher struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Regex bool   `json:"regex"`
}

// RuntimeUInt32
// Indicates that the route should additionally match on a runtime key
type RuntimeUInt32 struct {
	DefaultValue uint32 `json:"default_value"`
	RuntimeKey   string `json:"runtime_key"`
}

// RouteAction
// Route request to some upstream clusters.
type RouteAction struct {
	ClusterName        string            `json:"cluster_name"`
	ClusterHeader      string            `json:"cluster_header"`
	TotalClusterWeight uint32            `json:"total_cluster_weight"`
	WeightedClusters   []WeightedCluster `json:"weighted_clusters"`
	MetadataMatch      Metadata          `json:"metadata_match"`
	Timeout            time.Duration     `json:"timeout"`
	RetryPolicy        *RetryPolicy      `json:"retry_policy"`
}

// WeightedCluster.
// Multiple upstream clusters unsupport stream filter type:  healthcheckcan be specified for a given route.
// The request is routed to one of the upstream
// clusters based on weights assigned to each cluster
type WeightedCluster struct {
	Cluster          ClusterWeight `json:"cluster"`
	RuntimeKeyPrefix string        `json:"runtime_key_prefix"` // not used currently
}

// ClusterWeight.
// clusters along with weights that indicate the percentage
// of traffic to be forwarded to each cluster
type ClusterWeight struct {
	Name          string   `json:"name"`
	Weight        uint32   `json:"weight"`
	MetadataMatch Metadata `json:"metadata_match"`
}

// FilterConfig is a config to make up a filter
// Type is the filter's type
type FilterConfig struct {
	Type   string                 `json:"type,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

type RetryPolicy struct {
	RetryOn      bool          `json:"retry_on"`
	RetryTimeout time.Duration `json:"retry_timeout"`
	NumRetries   uint32        `json:"num_retries"`
}

// AccessLogConfig for making up access log
type AccessLogConfig struct {
	LogPath   string `json:"log_path,omitempty"`
	LogFormat string `json:"log_format,omitempty"`
}

// ListenerConfig
// for making up a listener in mosn
type ListenerConfig struct {
	Name          string         `json:"name,omitempty"`
	Address       string         `json:"address,omitempty"`
	BindToPort    bool           `json:"bind_port"`
	Inspector     bool           `json:"inspector,omitempty"`
	FilterChains  []FilterChain  `json:"filter_chains"`
	StreamFilters []FilterConfig `json:"stream_filters,omitempty"`
	//logger
	LogPath  string `json:"log_path,omitempty"`
	LogLevel string `json:"log_level,omitempty"`
	//HandOffRestoredDestinationConnections
	HandOffRestoredDestinationConnections bool `json:"handoff_restoreddestination"`
	//access log
	AccessLogs []AccessLogConfig `json:"access_logs,omitempty"`
}

// TLSConfig
// Status is the switch to use tls or not
type TLSConfig struct {
	Status       bool                   `json:"status,omitempty"`
	Type         string                 `json:"type,omitempty"`
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
	ExtendVerify map[string]interface{} `json:"extend_verify, omitempty"`
}

// ServerConfig for making up server for mosn
type ServerConfig struct {
	//default logger
	DefaultLogPath  string `json:"default_log_path,omitempty"`
	DefaultLogLevel string `json:"default_log_level,omitempty"`

	//graceful shutdown config
	GracefulTimeout DurationConfig `json:"graceful_timeout"`

	//go processor number
	Processor int `json:"processor"`

	Listeners []ListenerConfig `json:"listeners,omitempty"`
}

// HostConfig
type HostConfig struct {
	Address  string   `json:"address,omitempty"`
	Hostname string   `json:"hostname,omitempty"`
	Weight   uint32   `json:"weight,omitempty"`
	MetaData Metadata `json:"metadata"`
}

// ClusterHealthCheckConfig for health checking
type ClusterHealthCheckConfig struct {
	Protocol           string         `json:"protocol"`
	Timeout            DurationConfig `json:"timeout"`
	Interval           DurationConfig `json:"interval"`
	IntervalJitter     DurationConfig `json:"interval_jitter"`
	HealthyThreshold   uint32         `json:"healthy_threshold"`
	UnhealthyThreshold uint32         `json:"unhealthy_threshold"`
	CheckPath          string         `json:"check_path,omitempty"`
	ServiceName        string         `json:"service_name,omitempty"`
}

// ClusterSpecConfig
// not used currently
type ClusterSpecConfig struct {
	Subscribes []SubscribeSpecConfig `json:"subscribe,omitempty"`
}

// SubscribeSpecConfig
type SubscribeSpecConfig struct {
	ServiceName string `json:"service_name,omitempty"`
}

// ClusterConfig for making a cluster for mosn
// Hosts are the hosts belong to this cluster
type ClusterConfig struct {
	Name                 string                   `json:"name"`
	Type                 string                   `json:"type"`
	SubType              string                   `json:"sub_type"`
	LbType               string                   `json:"lb_type"`
	MaxRequestPerConn    uint32                   `json:"max_request_per_conn"`
	ConnBufferLimitBytes uint32                   `json:"conn_buffer_limit_bytes"`
	CircuitBreakers      []*CircuitBreakerConfig  `json:"circuit_breakers"`
	HealthCheck          ClusterHealthCheckConfig `json:"health_check,omitempty"`
	ClusterSpecConfig    ClusterSpecConfig        `json:"spec,omitempty"` //	ClusterSpecConfig
	Hosts                []HostConfig             `json:"hosts,omitempty"`
	LBSubsetConfig       LBSubsetConfig           `json:"lb_subset_config"`
	TLS                  TLSConfig                `json:"tls_context,omitempty"`
}

type LBSubsetConfig struct {
	FallBackPolicy  uint8             `json:"fall_back_policy"`
	DefaultSubset   map[string]string `json:"default_subset"`
	SubsetSelectors [][]string        `json:"subset_selectors"`
}

// CircuitBreakerConfig for realizing circuit breaker for cluster
type CircuitBreakerConfig struct {
	Priority           string `json:"priority"`
	MaxConnections     uint32 `json:"max_connections"`
	MaxPendingRequests uint32 `json:"max_pending_requests"`
	MaxRequests        uint32 `json:"max_requests"`
	MaxRetries         uint32 `json:"max_retries"`
}

// ClusterManagerConfig for making up cluster manager
// Cluster is the global cluster of mosn
type ClusterManagerConfig struct {
	// Note: consider to use standard configure
	AutoDiscovery bool `json:"auto_discovery"`
	// Note: this is a hack method to realize cluster's  health check which push by registry
	RegistryUseHealthCheck bool            `json:"registry_use_health_check"`
	Clusters               []ClusterConfig `json:"clusters,omitempty"`
}

// ServiceRegistryConfig
// not used currently
type ServiceRegistryConfig struct {
	ServiceAppInfo ServiceAppInfoConfig   `json:"application"`
	ServicePubInfo []ServicePubInfoConfig `json:"publish_info,omitempty"`
}

// ServiceAppInfoConfig
type ServiceAppInfoConfig struct {
	AntShareCloud bool   `json:"ant_share_cloud"`
	DataCenter    string `json:"data_center,omitempty"`
	AppName       string `json:"app_name,omitempty"`
}

// ServicePubInfoConfig
type ServicePubInfoConfig struct {
	ServiceName string `json:"service_name,omitempty"`
	PubData     string `json:"pub_data,omitempty"`
}

// TCPRouteConfig
type TCPRouteConfig struct {
	Cluster          string   `json:"cluster,omitempty"`
	SourceAddrs      []string `json:"source_addrs,omitempty"`
	DestinationAddrs []string `json:"destination_addrs,omitempty"`
}

// TCPProxy
type TCPProxyConfig struct {
	Routes []TCPRouteConfig `json:"routes,omitempty"`
}

// MOSNConfig make up mosn to start the mosn project
// Servers contains the listener, filter and so on
// ClusterManager used to manage the upstream
type MOSNConfig struct {
	Servers         []ServerConfig        `json:"servers,omitempty"`         //server config
	ClusterManager  ClusterManagerConfig  `json:"cluster_manager,omitempty"` //cluster config
	ServiceRegistry ServiceRegistryConfig `json:"service_registry"`          //service registry config, used by service discovery module
	//tracing config
	RawDynamicResources jsoniter.RawMessage `json:"dynamic_resources,omitempty"` //dynamic_resources raw message
	RawStaticResources  jsoniter.RawMessage `json:"static_resources,omitempty"`  //static_resources raw message
}

// Mode is mosn's starting type
type Mode uint8

// File means start from config file
// Xds means start from xds
// Mix means start both from file and Xds
const (
	File Mode = iota
	Xds
	Mix
)

func (c *MOSNConfig) Mode() Mode {
	if len(c.Servers) > 0 {
		if len(c.RawStaticResources) == 0 || len(c.RawDynamicResources) == 0 {
			return File
		}

		return Mix
	} else if len(c.RawStaticResources) > 0 && len(c.RawDynamicResources) > 0 {
		return Xds
	}

	return File
}

// DurationConfig
// wrapper for time.Duration, so time config can be written in '300ms' or '1h' format
type DurationConfig struct {
	time.Duration
}

// UnmarshalJSON get DurationConfig.Duration from json file
func (d *DurationConfig) UnmarshalJSON(b []byte) (err error) {
	d.Duration, err = time.ParseDuration(strings.Trim(string(b), `"`))
	return
}

// MarshalJSON
func (d DurationConfig) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

// Load config file and parse
func Load(path string) *MOSNConfig {
	log.Println("load config from : ", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln("load config failed, ", err)
		os.Exit(1)
	}
	configPath, _ = filepath.Abs(path)
	err = json.Unmarshal(content, &config)

	if err != nil {
		log.Fatalln("json unmarshal config failed, ", err)
		os.Exit(1)
	}
	return &config
}
