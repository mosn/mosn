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

package configmanager

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/ghodss/yaml"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

var (
	// configPath stores the config file path
	configPath string
	// configLock controls the stored config
	configLock sync.RWMutex
	// conf keeps the mosn config
	conf effectiveConfig
	// configLoadFunc can be replaced by load config extension
	configLoadFunc ConfigLoadFunc = DefaultConfigLoad
)

// protetced configPath, read only
func GetConfigPath() string {
	return configPath
}

// ConfigLoadFunc parse a input(usually file path) into a mosn config
type ConfigLoadFunc func(path string) *v2.MOSNConfig

// RegisterConfigLoadFunc can replace a new config load function instead of default
func RegisterConfigLoadFunc(f ConfigLoadFunc) {
	configLoadFunc = f
}

func DefaultConfigLoad(path string) *v2.MOSNConfig {
	log.StartLogger.Infof("load config from :  %s", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.StartLogger.Fatalf("[config] [default load] load config failed, error: %v", err)
	}
	cfg := &v2.MOSNConfig{}
	if yamlFormat(path) {
		bytes, err := yaml.YAMLToJSON(content)
		if err != nil {
			log.StartLogger.Fatalf("[config] [default load] translate yaml to json error: %v", err)
		}
		content = bytes
	}
	// translate to lower case
	err = json.Unmarshal(content, cfg)
	if err != nil {
		log.StartLogger.Fatalf("[config] [default load] json unmarshal config failed, error: %v", err)
	}
	return cfg

}

// Load config file and parse
func Load(path string) *v2.MOSNConfig {
	configPath, _ = filepath.Abs(path)
	cfg := configLoadFunc(path)
	return cfg
}

func yamlFormat(path string) bool {
	ext := filepath.Ext(path)
	if ext == ".yaml" || ext == ".yml" {
		return true
	}
	return false
}
