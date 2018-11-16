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
	Protocol             string         `json:"protocol"`
	TimeoutConfig        DurationConfig `json:"timeout"`
	IntervalConfig       DurationConfig `json:"interval"`
	IntervalJitterConfig DurationConfig `json:"interval_jitter"`
	HealthyThreshold     uint32         `json:"healthy_threshold"`
	UnhealthyThreshold   uint32         `json:"unhealthy_threshold"`
	CheckPath            string         `json:"check_path,omitempty"`
	ServiceName          string         `json:"service_name,omitempty"`
}

type HostConfig struct {
	Address        string         `json:"address,omitempty"`
	Hostname       string         `json:"hostname,omitempty"`
	Weight         uint32         `json:"weight,omitempty"`
	MetaDataConfig MetadataConfig `json:"metadata,omitempty"`
	TLSDisable     bool           `json:"tls_disable,omitempty"`
}

type ListenerConfig struct {
	Name                                  string        `json:"name"`
	AddrConfig                            string        `json:"address"`
	BindToPort                            bool          `json:"bind_port"`
	HandOffRestoredDestinationConnections bool          `json:"handoff_restoreddestination"`
	LogPath                               string        `json:"log_path,omitempty"`
	LogLevelConfig                        string        `json:"log_level,omitempty"`
	AccessLogs                            []AccessLog   `json:"access_logs,omitempty"`
	FilterChains                          []FilterChain `json:"filter_chains"` // only one filterchains at this time
	StreamFilters                         []Filter      `json:"stream_filters,omitempty"`
	Inspector                             bool          `json:"inspector,omitempty"`
}

type TCPRouteConfig struct {
	Cluster string   `json:"cluster,omitempty"`
	Sources []string `json:"source_addrs,omitempty"`
	Dests   []string `json:"destination_addrs,omitempty"`
}

type HealthCheckFilterConfig struct {
	PassThrough                 bool               `json:"passthrough"`
	CacheTimeConfig             DurationConfig     `json:"cache_time"`
	Endpoint                    string             `json:"endpoint"`
	ClusterMinHealthyPercentage map[string]float32 `json:"cluster_min_healthy_percentages"`
}

type FaultInjectConfig struct {
	DelayPercent        uint32         `json:"delay_percent"`
	DelayDurationConfig DurationConfig `json:"delay_duration"`
}

type RouterConfig struct {
	Match           RouterMatch            `json:"match"`
	Route           RouteAction            `json:"route"`
	Redirect        RedirectAction         `json:"redirect"`
	MetadataConfig  MetadataConfig         `json:"metadata"`
	Decorator       Decorator              `json:"decorator"`
	PerFilterConfig map[string]interface{} `json:"per_filter_config"`
}

type RouterActionConfig struct {
	ClusterName             string               `json:"cluster_name"`
	ClusterHeader           string               `json:"cluster_header"`
	WeightedClusters        []WeightedCluster    `json:"weighted_clusters"`
	MetadataConfig          MetadataConfig       `json:"metadata_match"`
	TimeoutConfig           DurationConfig       `json:"timeout"`
	RetryPolicy             *RetryPolicy         `json:"retry_policy"`
	PrefixRewrite           string               `json:"prefix_rewrite"`
	HostRewrite             string               `json:"host_rewrite"`
	AutoHostRewrite         bool                 `json:"auto_host_rewrite"`
	RequestHeadersToAdd     []*HeaderValueOption `json:"request_headers_to_add"`
	ResponseHeadersToAdd    []*HeaderValueOption `json:"response_headers_to_add"`
	ResponseHeadersToRemove []string             `json:"response_headers_to_remove"`
}

type ClusterWeightConfig struct {
	Name           string         `json:"name"`
	Weight         uint32         `json:"weight"`
	MetadataConfig MetadataConfig `json:"metadata_match"`
}

type RetryPolicyConfig struct {
	RetryOn            bool           `json:"retry_on"`
	RetryTimeoutConfig DurationConfig `json:"retry_timeout"`
	NumRetries         uint32         `json:"num_retries"`
}
