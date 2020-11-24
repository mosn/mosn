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

package streamfilter

import (
	"encoding/json"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"path/filepath"
)

// StreamFiltersConfig is the stream filter config array.
type StreamFiltersConfig = []v2.Filter

// StreamFilters is the stream filter config attached with a name.
type StreamFilters struct {
	Name    string              `json:"name,omitempty"`
	Filters StreamFiltersConfig `json:"stream_filters,omitempty"`
}

// LoadAndRegisterStreamFilters load and register stream filter config from file.
func LoadAndRegisterStreamFilters(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.DefaultLogger.Errorf("[streamfilter] LoadAndRegisterStreamFilters fail to resolve abs path, err: %v", err)
		return
	}
	content, err := ioutil.ReadFile(absPath)
	if err != nil {
		log.DefaultLogger.Errorf("[streamfilter] LoadAndRegisterStreamFilters load config failed, error: %v", err)
		return
	}
	fileExt := filepath.Ext(path)
	if fileExt == ".yaml" || fileExt == ".yml" {
		bytes, err := yaml.YAMLToJSON(content)
		if err != nil {
			log.DefaultLogger.Errorf("[streamfilter] LoadAndRegisterStreamFilters translate yaml to json error: %v", err)
			return
		}
		content = bytes
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[streamfilter] LoadAndRegisterStreamFilters load config from file: %v", absPath)
	}
	ParseAndRegisterStreamFilters(content)
}

// ParseAndRegisterStreamFilters parse and register stream filter config from raw bytes.
func ParseAndRegisterStreamFilters(config []byte) {
	type fullConfig struct {
		SF []StreamFilters `json:"stream_filters,omitempty"`
	}
	var mc fullConfig
	err := json.Unmarshal(config, &mc)
	if err != nil {
		log.DefaultLogger.Errorf("[streamfilter] ParseAndRegisterStreamFilters unmarshal err: %v", err)
		return
	}
	RegisterStreamFilters(mc.SF)
}

// RegisterStreamFilters register stream filter from config.
func RegisterStreamFilters(configs []StreamFilters) {
	for _, config := range configs {
		if config.Name == "" {
			log.DefaultLogger.Errorf("[streamfilter] RegisterStreamFilters empty name")
			continue
		}
		if config.Filters == nil || len(config.Filters) <= 0 {
			log.DefaultLogger.Errorf("[streamfilter] RegisterStreamFilters empty filters")
			continue
		}
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[streamfilter] RegisterStreamFilters name: %v, config: %v", config.Name, config.Filters)
		}
		err := GetStreamFilterManager().AddOrUpdateStreamFilterConfig(config.Name, config.Filters)
		if err != nil {
			log.DefaultLogger.Errorf("[streamfilter] RegisterStreamFilters internal err: %v", err)
		}
	}
}

// GetStreamFilters returns a stream filter factory by filter.Type.
func GetStreamFilters(configs []v2.Filter) []api.StreamFilterChainFactory {
	var factories []api.StreamFilterChainFactory

	for _, c := range configs {
		sfcc, err := api.CreateStreamFilterChainFactory(c.Type, c.Config)
		if err != nil {
			log.DefaultLogger.Errorf("[config] get stream filter failed, type: %s, error: %v", c.Type, err)
			continue
		}
		if sfcc != nil {
			factories = append(factories, sfcc)
		}
	}

	return factories
}