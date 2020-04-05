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
	"time"

	"istio.io/api/mixer/v1/config/client"
	"mosn.io/api"
)

type HealthCheckFilterConfig struct {
	PassThrough                 bool               `json:"passthrough,omitempty"`
	CacheTimeConfig             api.DurationConfig `json:"cache_time,omitempty"`
	Endpoint                    string             `json:"endpoint,omitempty"`
	ClusterMinHealthyPercentage map[string]float32 `json:"cluster_min_healthy_percentages,omitempty"`
}

type FaultInjectConfig struct {
	DelayPercent        uint32             `json:"delay_percent,omitempty"`
	DelayDurationConfig api.DurationConfig `json:"delay_duration,omitempty"`
}

type DelayInjectConfig struct {
	Percent             uint32             `json:"percentage,omitempty"`
	DelayDurationConfig api.DurationConfig `json:"fixed_delay,omitempty"`
}

type FaultToleranceFilterConfig struct {
	Effective             bool
	Kind                  string
	Protocol              []string
	ExceptionType         []string
	TimeWindow            uint32
	LeastWindowCount      uint64
	MaxHostCount          uint64
	MaxHostRatio          float32
	ExceptionRateMultiple float64
	StateDimension        []string
}

type FaultToleranceKind string

const (
	OFF     FaultToleranceKind = "OFF"
	MONITOR FaultToleranceKind = "MONITOR"
	ON      FaultToleranceKind = "ON"
)

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
	MIXER          = "mixer"
	FaultStream    = "fault"
	PayloadLimit   = "payload_limit"
	FaultTolerance = "fault_tolerance"
)

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

// PayloadLimitInject
type StreamPayloadLimit struct {
	MaxEntitySize int32 `json:"max_entity_size "`
	HttpStatus    int32 `json:"http_status"`
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
