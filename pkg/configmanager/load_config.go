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
	"log"
	"path/filepath"
	"sync"

	"github.com/ghodss/yaml"
	v2 "mosn.io/mosn/pkg/config/v2"
)

var (
	configPath     string
	configLock     sync.Mutex
	config         v2.MOSNConfig
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
	log.Println("load config from : ", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln("[config] [default load] load config failed, ", err)
	}
	cfg := &v2.MOSNConfig{}
	// translate to lower case
	err = json.Unmarshal(content, cfg)
	if err != nil {
		log.Fatalln("[config] [default load] json unmarshal config failed, ", err)
	}
	return cfg

}

func YAMLConfigLoad(path string) *v2.MOSNConfig {
	log.Println("load config in YAML format from : ", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln("[config] [yaml load] load config failed, ", err)
	}
	cfg := &v2.MOSNConfig{}

	bytes, err := yaml.YAMLToJSON(content)
	if err != nil {
		log.Fatalln("[config] [yaml load] convert YAML to JSON failed, ", err)
	}

	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		log.Fatalln("[config] [yaml load] yaml unmarshal config failed, ", err)
	}
	return cfg

}

// Load config file and parse
func Load(path string) *v2.MOSNConfig {
	configPath, _ = filepath.Abs(path)
	if yamlFormat(path) {
		RegisterConfigLoadFunc(YAMLConfigLoad)
	}
	if cfg := configLoadFunc(path); cfg != nil {
		config = *cfg
	}
	return &config
}

func yamlFormat(path string) bool {
	ext := filepath.Ext(path)
	if ext == ".yaml" || ext == ".yml" {
		return true
	}
	return false
}
