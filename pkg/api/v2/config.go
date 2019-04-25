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

type HealthCheckConfig struct {
	Protocol             string                 `json:"protocol,omitempty"`
	TimeoutConfig        DurationConfig         `json:"timeout,omitempty"`
	IntervalConfig       DurationConfig         `json:"interval,omitempty"`
	IntervalJitterConfig DurationConfig         `json:"interval_jitter,omitempty"`
	HealthyThreshold     uint32                 `json:"healthy_threshold,omitempty"`
	UnhealthyThreshold   uint32                 `json:"unhealthy_threshold,omitempty"`
	ServiceName          string                 `json:"service_name,omitempty"`
	SessionConfig        map[string]interface{} `json:"check_config,omitempty"`
	CommonCallbacks      []string               `json:"common_callbacks,omitempty"` // HealthCheck support register some common callbacks that are not related to specific cluster
}

type HostConfig struct {
	Address        string          `json:"address,omitempty"`
	Hostname       string          `json:"hostname,omitempty"`
	Weight         uint32          `json:"weight,omitempty"`
	MetaDataConfig *MetadataConfig `json:"metadata,omitempty"`
	TLSDisable     bool            `json:"tls_disable,omitempty"`
}

// ListenerType: Ingress or Egress
type ListenerType string

const EGRESS ListenerType = "egress"
const INGRESS ListenerType = "ingress"

type ListenerConfig struct {
	Name                                  string        `json:"name,omitempty"`
	Type                                  ListenerType  `json:"type,omitempty"`
	AddrConfig                            string        `json:"address,omitempty"`
	BindToPort                            bool          `json:"bind_port,omitempty"`
	HandOffRestoredDestinationConnections bool          `json:"handoff_restoreddestination,omitemptY"`
	AccessLogs                            []AccessLog   `json:"access_logs,omitempty"`
	FilterChains                          []FilterChain `json:"filter_chains,omitempty"` // only one filterchains at this time
	StreamFilters                         []Filter      `json:"stream_filters,omitempty"`
	Inspector                             bool          `json:"inspector,omitempty"`
}

type TCPRouteConfig struct {
	Cluster string   `json:"cluster,omitempty"`
	Sources []string `json:"source_addrs,omitempty"`
	Dests   []string `json:"destination_addrs,omitempty"`
}

type HealthCheckFilterConfig struct {
	PassThrough                 bool               `json:"passthrough,omitempty"`
	CacheTimeConfig             DurationConfig     `json:"cache_time,omitempty"`
	Endpoint                    string             `json:"endpoint,omitempty"`
	ClusterMinHealthyPercentage map[string]float32 `json:"cluster_min_healthy_percentages,omitempty"`
}

type FaultInjectConfig struct {
	DelayPercent        uint32         `json:"delay_percent,omitempty"`
	DelayDurationConfig DurationConfig `json:"delay_duration,omitempty"`
}

type DelayInjectConfig struct {
	Percent             uint32         `json:"percentage,omitempty"`
	DelayDurationConfig DurationConfig `json:"fixed_delay,omitempty"`
}

type RouterConfigurationConfig struct {
	RouterConfigName        string               `json:"router_config_name,omitempty"`
	RequestHeadersToAdd     []*HeaderValueOption `json:"request_headers_to_add,omitempty"`
	ResponseHeadersToAdd    []*HeaderValueOption `json:"response_headers_to_add,omitempty"`
	ResponseHeadersToRemove []string             `json:"response_headers_to_remove,omitempty"`
	RouterConfigPath        string               `json:"router_configs, omitempty"`
	StaticVirtualHosts      []*VirtualHost       `json:"virtual_hosts,omitempty"`
}

type RouterConfig struct {
	Match           RouterMatch            `json:"match,omitempty"`
	Route           RouteAction            `json:"route,omitempty"`
	DirectResponse  *DirectResponseAction  `json:"direct_response,omitempty"`
	MetadataConfig  *MetadataConfig        `json:"metadata,omitempty"`
	PerFilterConfig map[string]interface{} `json:"per_filter_config,omitempty"`
}

type RouterActionConfig struct {
	ClusterName             string               `json:"cluster_name,omitempty"`
	UpstreamProtocol        string               `json:"upstream_protocol,omitempty"`
	ClusterHeader           string               `json:"cluster_header,omitempty"`
	WeightedClusters        []WeightedCluster    `json:"weighted_clusters,omitempty"`
	MetadataConfig          *MetadataConfig      `json:"metadata_match,omitempty"`
	TimeoutConfig           DurationConfig       `json:"timeout,omitempty"`
	RetryPolicy             *RetryPolicy         `json:"retry_policy,omitempty"`
	PrefixRewrite           string               `json:"prefix_rewrite,omitempty"`
	HostRewrite             string               `json:"host_rewrite,omitempty"`
	AutoHostRewrite         bool                 `json:"auto_host_rewrite,omitempty"`
	RequestHeadersToAdd     []*HeaderValueOption `json:"request_headers_to_add,omitempty"`
	ResponseHeadersToAdd    []*HeaderValueOption `json:"response_headers_to_add,omitempty"`
	ResponseHeadersToRemove []string             `json:"response_headers_to_remove,omitempty"`
}

type ClusterWeightConfig struct {
	Name           string          `json:"name,omitempty"`
	Weight         uint32          `json:"weight,omitempty"`
	MetadataConfig *MetadataConfig `json:"metadata_match,omitempty"`
}

type RetryPolicyConfig struct {
	RetryOn            bool           `json:"retry_on,omitempty"`
	RetryTimeoutConfig DurationConfig `json:"retry_timeout,omitempty"`
	NumRetries         uint32         `json:"num_retries,omitempty"`
}

type FilterChainConfig struct {
	FilterChainMatch string      `json:"match,omitempty"`
	TLSConfig        *TLSConfig  `json:"tls_context,omitempty"`
	TLSConfigs       []TLSConfig `json:"tls_context_set,omitempty"`
	Filters          []Filter    `json:"filters,omitempty"`
}
