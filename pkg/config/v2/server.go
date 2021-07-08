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
	"errors"
	"net"
	"strings"

	"mosn.io/api"
)

// ServerConfig for making up server for mosn
type ServerConfig struct {
	//default logger
	ServerName      string `json:"mosn_server_name,omitempty"`
	DefaultLogPath  string `json:"default_log_path,omitempty"`
	DefaultLogLevel string `json:"default_log_level,omitempty"`
	GlobalLogRoller string `json:"global_log_roller,omitempty"`

	UseNetpollMode bool `json:"use_netpoll_mode,omitempty"`
	//graceful shutdown config
	GracefulTimeout api.DurationConfig `json:"graceful_timeout,omitempty"`
	// OptimizeLocalWrite set to true means if a connection remote address is
	// localhost, we will use a goroutine for write, which can get better performance
	// but lower write time costs accuracy.
	OptimizeLocalWrite bool `json:"optimize_local_write,omitempty"`

	// int go processor number
	// string set auto means use real cpu core or limit cpu core
	Processor interface{} `json:"processor,omitempty"`

	Listeners []Listener `json:"listeners,omitempty"`

	Routers []*RouterConfiguration `json:"routers,omitempty"`
}

// ListenerType: Ingress or Egress
type ListenerType string

const EGRESS ListenerType = "egress"
const INGRESS ListenerType = "ingress"

type ListenerConfig struct {
	Name                  string              `json:"name,omitempty"`
	Type                  ListenerType        `json:"type,omitempty"`
	AddrConfig            string              `json:"address,omitempty"`
	BindToPort            bool                `json:"bind_port,omitempty"`
	Network               string              `json:"network,omitempty"`
	UseOriginalDst        bool                `json:"use_original_dst,omitempty"`
	AccessLogs            []AccessLog         `json:"access_logs,omitempty"`
	ListenerFilters       []Filter            `json:"listener_filters,omitempty"`
	FilterChains          []FilterChain       `json:"filter_chains,omitempty"` // only one filterchains at this time
	StreamFilters         []Filter            `json:"stream_filters,omitempty"`
	Inspector             bool                `json:"inspector,omitempty"`
	ConnectionIdleTimeout *api.DurationConfig `json:"connection_idle_timeout,omitempty"`
	TunnelListener      bool                  `json:"tunnel_listener,omitempty"`
}

// Listener contains the listener's information
type Listener struct {
	ListenerConfig
	Addr                    net.Addr        `json:"-"`
	ListenerTag             uint64          `json:"-"`
	ListenerScope           string          `json:"-"`
	PerConnBufferLimitBytes uint32          `json:"-"` // do not support config
	InheritListener         net.Listener    `json:"-"`
	InheritPacketConn       *net.PacketConn `json:"-"`
	Remain                  bool            `json:"-"`
}

func (l Listener) MarshalJSON() (b []byte, err error) {
	if l.Addr != nil {
		l.AddrConfig = l.Addr.String()
	}
	return json.Marshal(l.ListenerConfig)
}

var (
	ErrNoAddrListener   = errors.New("address is required in listener config")
	ErrUnsupportNetwork = errors.New("listener network only support tcp/udp/unix")
)

const defaultBufferLimit = 1 << 15

func (l *Listener) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &l.ListenerConfig); err != nil {
		return err
	}
	if l.AddrConfig == "" {
		return ErrNoAddrListener
	}
	if l.Network == "" {
		l.Network = "tcp" // default is tcp
	}
	l.Network = strings.ToLower(l.Network)
	var err error
	var addr net.Addr
	switch l.Network {
	case "udp":
		addr, err = net.ResolveUDPAddr("udp", l.AddrConfig)
	case "unix":
		addr, err = net.ResolveUnixAddr("unix", l.AddrConfig)
	case "tcp":
		addr, err = net.ResolveTCPAddr("tcp", l.AddrConfig)
	default: // only support tcp,udp,unix
		err = ErrUnsupportNetwork
	}
	if err != nil {
		return err
	}
	l.Addr = addr
	l.PerConnBufferLimitBytes = defaultBufferLimit
	return nil
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

type StreamFilterGoPluginConfig struct {
	SoPath        string `json:"so_path"`
	FactoryMethod string `json:"factory_method"`
}

// Filter is a config to make up a filter
type Filter struct {
	Type           string                      `json:"type,omitempty"`
	GoPluginConfig *StreamFilterGoPluginConfig `json:"go_plugin_config"`
	Config         map[string]interface{}      `json:"config,omitempty"`
}

type FilterChainConfig struct {
	FilterChainMatch string      `json:"match,omitempty"`
	TLSConfig        *TLSConfig  `json:"tls_context,omitempty"`
	TLSConfigs       []TLSConfig `json:"tls_context_set,omitempty"`
	Filters          []Filter    `json:"filters,omitempty"`
}
