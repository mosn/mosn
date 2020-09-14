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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"mosn.io/api"
	"mosn.io/pkg/utils"
)

type RouterConfigurationConfig struct {
	RouterConfigName        string               `json:"router_config_name,omitempty"`
	RequestHeadersToAdd     []*HeaderValueOption `json:"request_headers_to_add,omitempty"`
	ResponseHeadersToAdd    []*HeaderValueOption `json:"response_headers_to_add,omitempty"`
	ResponseHeadersToRemove []string             `json:"response_headers_to_remove,omitempty"`
	RouterConfigPath        string               `json:"router_configs,omitempty"`
	StaticVirtualHosts      []*VirtualHost       `json:"virtual_hosts,omitempty"`
}

type RouterConfig struct {
	Match                 RouterMatch            `json:"match,omitempty"`
	Route                 RouteAction            `json:"route,omitempty"`
	Redirect              *RedirectAction        `json:"redirect,omitempty"`
	DirectResponse        *DirectResponseAction  `json:"direct_response,omitempty"`
	MetadataConfig        *MetadataConfig        `json:"metadata,omitempty"`
	PerFilterConfig       map[string]interface{} `json:"per_filter_config,omitempty"`
	RequestMirrorPolicies *RequestMirrorPolicy   `json:"request_mirror_policies,omitempty"`
}

type RouterActionConfig struct {
	ClusterName             string               `json:"cluster_name,omitempty"`
	UpstreamProtocol        string               `json:"upstream_protocol,omitempty"`
	ClusterHeader           string               `json:"cluster_header,omitempty"`
	WeightedClusters        []WeightedCluster    `json:"weighted_clusters,omitempty"`
	HashPolicy              []HashPolicy         `json:"hash_policy,omitempty"`
	MetadataConfig          *MetadataConfig      `json:"metadata_match,omitempty"`
	TimeoutConfig           api.DurationConfig   `json:"timeout,omitempty"`
	RetryPolicy             *RetryPolicy         `json:"retry_policy,omitempty"`
	PrefixRewrite           string               `json:"prefix_rewrite,omitempty"`
	RegexRewrite            RegexRewrite         `json:"regex_rewrite,omitempty"`
	HostRewrite             string               `json:"host_rewrite,omitempty"`
	AutoHostRewrite         bool                 `json:"auto_host_rewrite,omitempty"`
	AutoHostRewriteHeader   string               `json:"auto_host_rewrite_header,omitempty"`
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
	RetryOn            bool               `json:"retry_on,omitempty"`
	RetryTimeoutConfig api.DurationConfig `json:"retry_timeout,omitempty"`
	NumRetries         uint32             `json:"num_retries,omitempty"`
}

// RegexRewrite represents the regex rewrite parameters
type RegexRewrite struct {
	Pattern      PatternConfig `json:"pattern,omitempty"`
	Substitution string        `json:"substitution,omitempty"`
}

type PatternConfig struct {
	GoogleRe2 GoogleRe2Config `json:"google_re2,omitempty"`
	Regex     string          `json:"regex,omitempty"`
}

// TODO: not implement yet
type GoogleRe2Config struct {
	MaxProgramSize uint32 `json:"max_program_size,omitempty"`
}

// Router, the list of routes that will be matched, in order, for incoming requests.
// The first route that matches will be used.
type Router struct {
	RouterConfig
	// Metadata is created from MetadataConfig, which is used to subset
	Metadata api.Metadata `json:"-"`
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
	MetadataMatch api.Metadata  `json:"-"`
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
	MetadataMatch api.Metadata `json:"-"`
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

// RouterConfiguration is a config for routers
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
	os.MkdirAll(rc.RouterConfigPath, 0755)
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
		if len(fileName) > MaxFilePath {
			fileName = fileName[:MaxFilePath]
		}
		fileName = strings.ReplaceAll(fileName, sep, "_")
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
			vh := &VirtualHost{}
			e := utils.ReadJsonFile(fileName, vh)
			switch e {
			case nil:
				rc.VirtualHosts = append(rc.VirtualHosts, vh)
			case utils.ErrIgnore:
				// do nothing
			default:
				return e
			}
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

// RedirectAction represents the redirect response parameters
type RedirectAction struct {
	ResponseCode   int    `json:"response_code,omitempty"`
	PathRedirect   string `json:"path_redirect,omitempty"`
	HostRedirect   string `json:"host_redirect,omitempty"`
	SchemeRedirect string `json:"scheme_redirect,omitempty"`
}

// DirectResponseAction represents the direct response parameters
type DirectResponseAction struct {
	StatusCode int    `json:"status,omitempty"`
	Body       string `json:"body,omitempty"`
}

// WeightedCluster ...
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

// Stream Proxy Route
type StreamRouteConfig struct {
	Cluster string   `json:"cluster,omitempty"`
	Sources []string `json:"source_addrs,omitempty"`
	Dests   []string `json:"destination_addrs,omitempty"`
}

// StreamRoute ...
type StreamRoute struct {
	Cluster          string
	SourceAddrs      []CidrRange
	DestinationAddrs []CidrRange
	SourcePort       string
	DestinationPort  string
}

// CidrRange ...
type CidrRange struct {
	Address string
	Length  uint32
	IpNet   *net.IPNet
}

// RequestMirrorPolicy mirror policy
type RequestMirrorPolicy struct {
	Cluster      string `json:"cluster,omitempty"`
	Percent      uint32 `json:"percent,omitempty"`
	TraceSampled bool   `json:"trace_sampled,omitempty"` // TODO not implement
}
