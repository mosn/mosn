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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"istio.io/api/mixer/v1/config/client"
	"sofastack.io/sofa-mosn/pkg/utils"
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
)

// Stream Filter's Type
const (
	MIXER       = "mixer"
	FaultStream = "fault"
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

// Cluster represents a cluster's information
type Cluster struct {
	Name                 string          `json:"name,omitempty"`
	ClusterType          ClusterType     `json:"type,omitempty"`
	SubType              string          `json:"sub_type,omitempty"` //not used yet
	LbType               LbType          `json:"lb_type,omitempty"`
	MaxRequestPerConn    uint32          `json:"max_request_per_conn,omitempty"`
	ConnBufferLimitBytes uint32          `json:"conn_buffer_limit_bytes,omitempty"`
	CirBreThresholds     CircuitBreakers `json:"circuit_breakers,omitempty"`
	HealthCheck          HealthCheck     `json:"health_check,omitempty"`
	Spec                 ClusterSpecInfo `json:"spec,omitempty"`
	LBSubSetConfig       LBSubsetConfig  `json:"lb_subset_config,omitempty"`
	TLS                  TLSConfig       `json:"tls_context,omitempty"`
	Hosts                []Host          `json:"hosts,omitempty"`
}

// HealthCheck is a configuration of health check
// use DurationConfig to parse string to time.Duration
type HealthCheck struct {
	HealthCheckConfig
	Timeout        time.Duration `json:"-"`
	Interval       time.Duration `json:"-"`
	IntervalJitter time.Duration `json:"-"`
}

// Marshal implement a json.Marshaler
func (hc HealthCheck) MarshalJSON() (b []byte, err error) {
	hc.HealthCheckConfig.IntervalConfig.Duration = hc.Interval
	hc.HealthCheckConfig.IntervalJitterConfig.Duration = hc.IntervalJitter
	hc.HealthCheckConfig.TimeoutConfig.Duration = hc.Timeout
	return json.Marshal(hc.HealthCheckConfig)
}

func (hc *HealthCheck) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &hc.HealthCheckConfig); err != nil {
		return err
	}
	hc.Timeout = hc.TimeoutConfig.Duration
	hc.Interval = hc.IntervalConfig.Duration
	hc.IntervalJitter = hc.IntervalJitterConfig.Duration
	return nil
}

// Host represenets a host information
type Host struct {
	HostConfig
	MetaData Metadata `json:"-"`
}

func (h Host) MarshalJSON() (b []byte, err error) {
	h.HostConfig.MetaDataConfig = metadataToConfig(h.MetaData)
	return json.Marshal(h.HostConfig)
}

func (h *Host) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &h.HostConfig); err != nil {
		return err
	}
	h.MetaData = configToMetadata(h.MetaDataConfig)
	return nil
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

func (hf HealthCheckFilter) MarshalJSON() (b []byte, err error) {
	hf.HealthCheckFilterConfig.CacheTimeConfig.Duration = hf.CacheTime
	return json.Marshal(hf.HealthCheckFilterConfig)
}

func (hf *HealthCheckFilter) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &hf.HealthCheckFilterConfig); err != nil {
		return err
	}
	hf.CacheTime = hf.CacheTimeConfig.Duration
	return nil
}

// FaultInject
type FaultInject struct {
	FaultInjectConfig
	DelayDuration uint64 `json:"-"`
}

func (f FaultInject) Marshal() (b []byte, err error) {
	f.FaultInjectConfig.DelayDurationConfig.Duration = time.Duration(f.DelayDuration)
	return json.Marshal(f.FaultInjectConfig)
}

func (f *FaultInject) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &f.FaultInjectConfig); err != nil {
		return err
	}
	f.DelayDuration = uint64(f.DelayDurationConfig.Duration)
	return nil
}

// StreamFaultInject
type StreamFaultInject struct {
	Delay           *DelayInject    `json:"delay,omitempty"`
	Abort           *AbortInject    `json:"abort,omitempty"`
	UpstreamCluster string          `json:"upstream_cluster,omitempty"`
	Headers         []HeaderMatcher `json:"headers,omitempty"`
}

type DelayInject struct {
	DelayInjectConfig
	Delay time.Duration `json:"-"`
}

func (d DelayInject) Marshal() (b []byte, err error) {
	d.DelayInjectConfig.DelayDurationConfig.Duration = d.Delay
	return json.Marshal(d.DelayInjectConfig)
}

func (d *DelayInject) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &d.DelayInjectConfig); err != nil {
		return err
	}
	d.Delay = d.DelayDurationConfig.Duration
	return nil
}

type AbortInject struct {
	Status  int    `json:"status,omitempty"`
	Percent uint32 `json:"percentage,omitempty"`
}

type Mixer struct {
	client.HttpClientConfig
}

// Router, the list of routes that will be matched, in order, for incoming requests.
// The first route that matches will be used.
type Router struct {
	RouterConfig
	// Metadata is created from MetadataConfig, which is used to subset
	Metadata Metadata `json:"-"`
}

func (r Router) MarshalJSON() (b []byte, err error) {
	r.RouterConfig.MetadataConfig = metadataToConfig(r.Metadata)
	return json.Marshal(r.RouterConfig)
}

func (r *Router) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &r.RouterConfig); err != nil {
		return err
	}
	r.Metadata = configToMetadata(r.MetadataConfig)
	return nil
}

// RouteAction represents the information of route request to upstream clusters
type RouteAction struct {
	RouterActionConfig
	MetadataMatch Metadata      `json:"-"`
	Timeout       time.Duration `json:"-"`
}

func (r RouteAction) MarshalJSON() (b []byte, err error) {
	r.RouterActionConfig.MetadataConfig = metadataToConfig(r.MetadataMatch)
	r.RouterActionConfig.TimeoutConfig.Duration = r.Timeout
	return json.Marshal(r.RouterActionConfig)
}

func (r *RouteAction) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &r.RouterActionConfig); err != nil {
		return err
	}
	r.Timeout = r.RouterActionConfig.TimeoutConfig.Duration
	r.MetadataMatch = configToMetadata(r.MetadataConfig)
	return nil
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

func (cw ClusterWeight) MarshalJSON() (b []byte, err error) {
	cw.ClusterWeightConfig.MetadataConfig = metadataToConfig(cw.MetadataMatch)
	return json.Marshal(cw.ClusterWeightConfig)
}

func (cw *ClusterWeight) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &cw.ClusterWeightConfig); err != nil {
		return err
	}
	cw.MetadataMatch = configToMetadata(cw.MetadataConfig)
	return nil
}

// RetryPolicy represents the retry parameters
type RetryPolicy struct {
	RetryPolicyConfig
	RetryTimeout time.Duration `json:"-"`
}

func (rp RetryPolicy) MarshalJSON() (b []byte, err error) {
	rp.RetryPolicyConfig.RetryTimeoutConfig.Duration = rp.RetryTimeout
	return json.Marshal(rp.RetryPolicyConfig)
}

func (rp *RetryPolicy) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &rp.RetryPolicyConfig); err != nil {
		return err
	}
	rp.RetryTimeout = rp.RetryTimeoutConfig.Duration
	return nil
}

// CircuitBreakers is a configuration of circuit breakers
// CircuitBreakers implements json.Marshaler and json.Unmarshaler
type CircuitBreakers struct {
	Thresholds []Thresholds
}

// CircuitBreakers's implements json.Marshaler and json.Unmarshaler
func (cb CircuitBreakers) MarshalJSON() (b []byte, err error) {
	return json.Marshal(cb.Thresholds)
}
func (cb *CircuitBreakers) UnmarshalJSON(b []byte) (err error) {
	return json.Unmarshal(b, &cb.Thresholds)
}

type Thresholds struct {
	MaxConnections     uint32 `json:"max_connections,omitempty"`
	MaxPendingRequests uint32 `json:"max_pending_requests,omitempty"`
	MaxRequests        uint32 `json:"max_requests,omitempty"`
	MaxRetries         uint32 `json:"max_retries,omitempty"`
}

// ClusterSpecInfo is a configuration of subscribe
type ClusterSpecInfo struct {
	Subscribes []SubscribeSpec `json:"subscribe,omitempty"`
}

// SubscribeSpec describes the subscribe server
type SubscribeSpec struct {
	Subscriber  string `json:"subscriber,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
}

// LBSubsetConfig is a configuration of load balance subset
type LBSubsetConfig struct {
	FallBackPolicy  uint8             `json:"fall_back_policy,omitempty"`
	DefaultSubset   map[string]string `json:"default_subset,omitempty"`
	SubsetSelectors [][]string        `json:"subset_selectors,omitempty"`
}

// TLSConfig is a configuration of tls context
type TLSConfig struct {
	Status            bool                   `json:"status,omitempty"`
	Type              string                 `json:"type,omitempty"`
	ServerName        string                 `json:"server_name,omitempty"`
	CACert            string                 `json:"ca_cert,omitempty"`
	CertChain         string                 `json:"cert_chain,omitempty"`
	PrivateKey        string                 `json:"private_key,omitempty"`
	VerifyClient      bool                   `json:"verify_client,omitempty"`
	RequireClientCert bool                   `json:"require_client_cert,omitempty"`
	InsecureSkip      bool                   `json:"insecure_skip,omitempty"`
	CipherSuites      string                 `json:"cipher_suites,omitempty"`
	EcdhCurves        string                 `json:"ecdh_curves,omitempty"`
	MinVersion        string                 `json:"min_version,omitempty"`
	MaxVersion        string                 `json:"max_version,omitempty"`
	ALPN              string                 `json:"alpn,omitempty"`
	Ticket            string                 `json:"ticket,omitempty"`
	Fallback          bool                   `json:"fall_back,omitempty"`
	ExtendVerify      map[string]interface{} `json:"extend_verify,omitempty"`
	SdsConfig         *SdsConfig             `json:"sds_source,omitempty"`
}

type SdsConfig struct {
	CertificateConfig *auth.SdsSecretConfig
	ValidationConfig  *auth.SdsSecretConfig
}

// Valid checks the whether the SDS Config is valid or not
func (c *SdsConfig) Valid() bool {
	return c != nil && c.CertificateConfig != nil && c.ValidationConfig != nil
}

// AccessLog for making up access log
type AccessLog struct {
	Path   string `json:"log_path,omitempty"`
	Format string `json:"log_format,omitempty"`
}

// FilterChain wraps a set of match criteria, an option TLS context,
// a set of filters, and various other parameters.
type FilterChain struct {
	FilterChainConfig
	TLSContexts []TLSConfig `json:"-"`
}

func (fc FilterChain) MarshalJSON() (b []byte, err error) {
	if len(fc.TLSContexts) > 0 { // use tls_context_set
		fc.TLSConfig = nil
		fc.TLSConfigs = fc.TLSContexts
	}
	return json.Marshal(fc.FilterChainConfig)
}

func (fc *FilterChain) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &fc.FilterChainConfig); err != nil {
		return err
	}
	if fc.TLSConfig != nil && len(fc.TLSConfigs) > 0 {
		return ErrDuplicateTLSConfig
	}
	if len(fc.TLSConfigs) > 0 {
		fc.TLSContexts = make([]TLSConfig, len(fc.TLSConfigs))
		copy(fc.TLSContexts, fc.TLSConfigs)
	} else { // no tls_context_set, use tls_context
		if fc.TLSConfig == nil { // no tls_context, generate a default one
			fc.TLSContexts = append(fc.TLSContexts, TLSConfig{})
		} else { // use tls_context
			fc.TLSContexts = append(fc.TLSContexts, *fc.TLSConfig)
		}
	}
	return nil
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
	Name               string                 `json:"name,omitempty"`
	DownstreamProtocol string                 `json:"downstream_protocol,omitempty"`
	UpstreamProtocol   string                 `json:"upstream_protocol,omitempty"`
	RouterConfigName   string                 `json:"router_config_name,omitempty"`
	ValidateClusters   bool                   `json:"validate_clusters,omitempty"`
	ExtendConfig       map[string]interface{} `json:"extend_config,omitempty"`
}

// HeaderValueOption is header name/value pair plus option to control append behavior.
type HeaderValueOption struct {
	Header *HeaderValue `json:"header,omitempty"`
	Append *bool        `json:"append,omitempty"`
}

// HeaderValue is header name/value pair.
type HeaderValue struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// RouterConfiguration is a filter for routers
// Filter type is:  "CONNECTION_MANAGER"
type RouterConfiguration struct {
	VirtualHosts []*VirtualHost `json:"-"`
	RouterConfigurationConfig
}

// Marshal memory config into json, if dynamic mode is configured, write json file
func (rc RouterConfiguration) MarshalJSON() (b []byte, err error) {
	if rc.RouterConfigPath == "" {
		rc.StaticVirtualHosts = rc.VirtualHosts
		return json.Marshal(rc.RouterConfigurationConfig)
	}
	// dynamic mode, should write file
	// first, get all the files in the directory
	files, err := ioutil.ReadDir(rc.RouterConfigPath)
	if err != nil {
		return nil, err
	}
	allFiles := make(map[string]struct{}, len(files))
	for _, f := range files {
		allFiles[f.Name()] = struct{}{}
	}
	// file name is virtualhost name, if not exists, use {unixnano}.json
	for _, vh := range rc.VirtualHosts {
		fileName := vh.Name
		if fileName == "" {
			fileName = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		data, err := json.MarshalIndent(vh, "", " ")
		if err != nil {
			return nil, err
		}
		fileName = fileName + ".json"
		delete(allFiles, fileName)
		fileName = path.Join(rc.RouterConfigPath, fileName)
		if err := utils.WriteFileSafety(fileName, data, 0644); err != nil {
			return nil, err
		}
	}
	// delete invalid files
	for f := range allFiles {
		os.Remove(path.Join(rc.RouterConfigPath, f))
	}
	return json.Marshal(rc.RouterConfigurationConfig)
}

func (rc *RouterConfiguration) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &rc.RouterConfigurationConfig); err != nil {
		return err
	}
	cfg := rc.RouterConfigurationConfig
	// only one of the config should be exists
	if len(cfg.StaticVirtualHosts) > 0 && cfg.RouterConfigPath != "" {
		return ErrDuplicateStaticAndDynamic
	}
	if len(cfg.StaticVirtualHosts) > 0 {
		rc.VirtualHosts = cfg.StaticVirtualHosts
	}
	// Traversing path and parse the json
	// assume all the files in the path are available json file, and no sub path
	if cfg.RouterConfigPath != "" {
		files, err := ioutil.ReadDir(cfg.RouterConfigPath)
		if err != nil {
			return err
		}
		for _, f := range files {
			fileName := path.Join(cfg.RouterConfigPath, f.Name())
			data, err := ioutil.ReadFile(fileName)
			if err != nil {
				return err
			}
			vh := &VirtualHost{}
			if err := json.Unmarshal(data, vh); err != nil {
				return err
			}
			rc.VirtualHosts = append(rc.VirtualHosts, vh)
		}
	}
	return nil
}

// VirtualHost is used to make up the route table
type VirtualHost struct {
	Name                    string               `json:"name,omitempty"`
	Domains                 []string             `json:"domains,omitempty"`
	Routers                 []Router             `json:"routers,omitempty"`
	RequireTLS              string               `json:"require_tls,omitempty"` // not used yet
	RequestHeadersToAdd     []*HeaderValueOption `json:"request_headers_to_add,omitempty"`
	ResponseHeadersToAdd    []*HeaderValueOption `json:"response_headers_to_add,omitempty"`
	ResponseHeadersToRemove []string             `json:"response_headers_to_remove,omitempty"`
}

// RouterMatch represents the route matching parameters
type RouterMatch struct {
	Prefix  string          `json:"prefix,omitempty"`  // Match request's Path with Prefix Comparing
	Path    string          `json:"path,omitempty"`    // Match request's Path with Exact Comparing
	Regex   string          `json:"regex,omitempty"`   // Match request's Path with Regex Comparing
	Headers []HeaderMatcher `json:"headers,omitempty"` // Match request's Headers
}

// DirectResponseAction represents the direct response parameters
type DirectResponseAction struct {
	StatusCode int    `json:"status,omitempty"`
	Body       string `json:"body,omitempty"`
}

// WeightedCluster.
// Multiple upstream clusters unsupport stream filter type:  healthcheckcan be specified for a given route.
// The request is routed to one of the upstream
// clusters based on weights assigned to each cluster
type WeightedCluster struct {
	Cluster ClusterWeight `json:"cluster,omitempty"`
}

// HeaderMatcher specifies a set of headers that the route should match on.
type HeaderMatcher struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
	Regex bool   `json:"regex,omitempty"`
}

// XProxyExtendConfig
type XProxyExtendConfig struct {
	SubProtocol string `json:"sub_protocol,omitempty"`
}

// ServiceRegistryInfo
type ServiceRegistryInfo struct {
	ServiceAppInfo ApplicationInfo     `json:"application,omitempty"`
	ServicePubInfo []PublishInfo       `json:"publish_info,omitempty"`
	MsgMetaInfo    map[string][]string `json:"msg_meta_info,omitempty"`
}

type ApplicationInfo struct {
	AntShareCloud bool   `json:"ant_share_cloud,omitempty"`
	DataCenter    string `json:"data_center,omitempty"`
	AppName       string `json:"app_name,omitempty"`
	Zone          string `json:"zone,omitempty"`
	DeployMode    bool   `json:"deploy_mode,omitempty"`
	MasterSystem  bool   `json:"master_system,omitempty"`
	CloudName     string `json:"cloud_name,omitempty"`
	HostMachine   string `json:"host_machine,omitempty"`
}

// PublishInfo implements json.Marshaler and json.Unmarshaler
type PublishInfo struct {
	Pub PublishContent
}

func (pb PublishInfo) MarshalJSON() (b []byte, err error) {
	return json.Marshal(pb.Pub)
}
func (pb *PublishInfo) UnmarshalJSON(b []byte) (err error) {
	return json.Unmarshal(b, &pb.Pub)
}

type PublishContent struct {
	ServiceName string `json:"service_name,omitempty"`
	PubData     string `json:"pub_data,omitempty"`
}

// StatsMatcher is a configuration for disabling stat instantiation.
// TODO: support inclusion_list
// TODO: support exclusion list/inclusion_list as pattern
type StatsMatcher struct {
	RejectAll       bool     `json:"reject_all,omitempty"`
	ExclusionLabels []string `json:"exclusion_labels,omitempty"`
	ExclusionKeys   []string `json:"exclusion_keys,omitempty"`
}

// ServerConfig for making up server for mosn
type ServerConfig struct {
	//default logger
	ServerName      string `json:"mosn_server_name,omitempty"`
	DefaultLogPath  string `json:"default_log_path,omitempty"`
	DefaultLogLevel string `json:"default_log_level,omitempty"`
	GlobalLogRoller string `json:"global_log_roller,omitempty"`

	UseNetpollMode bool `json:"use_netpoll_mode,omitempty"`
	//graceful shutdown config
	GracefulTimeout DurationConfig `json:"graceful_timeout,omitempty"`

	//go processor number
	Processor int `json:"processor,omitempty"`

	Listeners []Listener `json:"listeners,omitempty"`
}
