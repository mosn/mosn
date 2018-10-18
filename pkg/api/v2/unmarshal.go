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
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// MetadataConfig is a config for metadata
type MetadataConfig struct {
	MetaKey LbMeta `json:"filter_metadata"`
}
type LbMeta struct {
	LbMetaKey map[string]interface{} `json:"mosn.lb"`
}

// DurationConfig ia a wrapper for time.Duration, so time config can be written in '300ms' or '1h' format
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

// CircuitBreakers's implements json.Marshaler and json.Unmarshaler
func (cb CircuitBreakers) MarshalJSON() (b []byte, err error) {
	return json.Marshal(cb.Thresholds)
}
func (cb *CircuitBreakers) UnmarshalJSON(b []byte) (err error) {
	return json.Unmarshal(b, &cb.Thresholds)
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

func parseMetaData(cfg MetadataConfig) Metadata {
	result := Metadata{}
	mosnLb := cfg.MetaKey.LbMetaKey
	for k, v := range mosnLb {
		if vs, ok := v.(string); ok {
			result[k] = vs
		}
	}
	return result
}

func (h *Host) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &h.HostConfig); err != nil {
		return err
	}
	h.MetaData = parseMetaData(h.MetaDataConfig)
	return nil
}

func (hf *HealthCheckFilter) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &hf.HealthCheckFilterConfig); err != nil {
		return err
	}
	hf.CacheTime = hf.CacheTimeConfig.Duration
	return nil
}

func (f *FaultInject) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &f.FaultInjectConfig); err != nil {
		return err
	}
	f.DelayDuration = uint64(f.DelayDurationConfig.Duration)
	return nil
}

func (r *Router) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &r.RouterConfig); err != nil {
		return err
	}
	r.Metadata = parseMetaData(r.MetadataConfig)
	return nil
}

func (r *RouteAction) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &r.RouterActionConfig); err != nil {
		return err
	}
	r.Timeout = r.RouterActionConfig.TimeoutConfig.Duration
	r.MetadataMatch = parseMetaData(r.MetadataConfig)
	return nil
}

func (cw *ClusterWeight) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &cw.ClusterWeightConfig); err != nil {
		return err
	}
	cw.MetadataMatch = parseMetaData(cw.MetadataConfig)
	return nil
}

func (rp *RetryPolicy) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &rp.RetryPolicyConfig); err != nil {
		return err
	}
	rp.RetryTimeout = rp.RetryTimeoutConfig.Duration
	return nil
}

// PublishInfo's implements json.Marshaler and json.Unmarshaler
func (pb PublishInfo) MarshalJSON() (b []byte, err error) {
	return json.Marshal(pb.Pub)
}
func (pb *PublishInfo) UnmarshalJSON(b []byte) (err error) {
	return json.Unmarshal(b, &pb.Pub)
}
