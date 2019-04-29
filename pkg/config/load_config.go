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

package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
)

var (
	configPath     string
	config         MOSNConfig
	configLoadFunc ConfigLoadFunc = DefaultConfigLoad
)

// ConfigLoadFunc parse a input(usually file path) into a mosn config
type ConfigLoadFunc func(path string) *MOSNConfig

// RegisterConfigLoadFunc can replace a new config load function instead of default
func RegisterConfigLoadFunc(f ConfigLoadFunc) {
	configLoadFunc = f
}

func DefaultConfigLoad(path string) *MOSNConfig {
	log.Println("load config from : ", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln("load config failed, ", err)
	}
	cfg := &MOSNConfig{}
	// translate to lower case
	err = json.Unmarshal(content, cfg)
	if err != nil {
		log.Fatalln("json unmarshal config failed, ", err)
	}
	return cfg

}

// Load config file and parse
func Load(path string) *MOSNConfig {
	configPath, _ = filepath.Abs(path)
	if cfg := configLoadFunc(path); cfg != nil {
		config = *cfg
	}
	return &config
}
