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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/alipay/sofa-mosn/pkg/utils"
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

func parseMetaData(cfg *MetadataConfig) Metadata {
	result := Metadata{}
	if cfg != nil {
		mosnLb := cfg.MetaKey.LbMetaKey
		for k, v := range mosnLb {
			if vs, ok := v.(string); ok {
				result[k] = vs
			}
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
func (d *DelayInject) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &d.DelayInjectConfig); err != nil {
		return err
	}
	d.Delay = d.DelayDurationConfig.Duration
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

var ErrDuplicateTLSConfig = errors.New("tls_context and tls_context_set can only exists one at the same time")

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
func (fc FilterChain) MarshalJSON() (b []byte, err error) {
	if len(fc.TLSContexts) > 0 { // use tls_context_set
		fc.TLSConfig = nil
		fc.TLSConfigs = fc.TLSContexts
	}
	return json.Marshal(fc.FilterChainConfig)
}

var ErrDuplicateStaticAndDynamic = errors.New("only one of static config or dynamic config should be exists")

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
