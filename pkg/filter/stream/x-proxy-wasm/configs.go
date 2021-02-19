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

package x_proxy_wasm

import (
	"encoding/json"
	"errors"
	"runtime"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

type filterConfig struct {
	FromWasmPlugin string            `json:"from_wasm_plugin,omitempty"`
	VmConfig       *v2.WasmVmConfig  `json:"vm_config,omitempty"`
	InstanceNum    int               `json:"instance_num,omitempty"`
	RootContextID  int32             `json:"root_context_id, omitempty"`
	UserData       map[string]string `json:"-"`
}

func parseFilterConfig(cfg map[string]interface{}) (*filterConfig, error) {
	config := filterConfig{
		UserData: make(map[string]string),
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &config); err != nil {
		log.DefaultLogger.Errorf("[x-proxy-wasm][config] fail to unmarshal filter config, err: %v", err)
		return nil, err
	}

	if config.FromWasmPlugin != "" {
		config.VmConfig = nil
		config.InstanceNum = 0
	} else {
		if config.VmConfig == nil {
			log.DefaultLogger.Errorf("[x-proxy-wasm][config] fail to parse vm config")
			return nil, errors.New("fail to parse vm config")
		}
		if config.InstanceNum <= 0 {
			config.InstanceNum = runtime.NumCPU()
		}
	}

	m := make(map[string]interface{})

	if err := json.Unmarshal(data, &m); err != nil {
		log.DefaultLogger.Errorf("[x-proxy-wasm][config] fail to unmarshal user data, err: %v", err)
		return nil, err
	}

	for k, v := range m {
		if _, ok := v.(string); !ok {
			delete(m, k)
		}
	}
	delete(m, "from_wasm_plugin")
	if len(m) > 0 {
		config.UserData = make(map[string]string, len(m))
		for k, v := range m {
			config.UserData[k] = v.(string)
		}
	}

	return &config, nil
}
