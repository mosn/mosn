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

import "time"

// StreamProxy
type StreamProxy struct {
	StatPrefix         string         `json:"stat_prefix,omitempty"`
	Cluster            string         `json:"cluster,omitempty"`
	IdleTimeout        *time.Duration `json:"idle_timeout,omitempty"`
	MaxConnectAttempts uint32         `json:"max_connect_attempts,omitempty"`
	Routes             []*StreamRoute `json:"routes,omitempty"`
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

// XProxyExtendConfig
type XProxyExtendConfig struct {
	SubProtocol string `json:"sub_protocol,omitempty"`
}

// ProxyGeneralExtendConfig is a general config for proxy
type ProxyGeneralExtendConfig struct {
	Http2UseStream     bool `json:"http2_use_stream,omitempty"`
	MaxRequestBodySize int  `json:"max_request_body_size,omitempty"`
}
