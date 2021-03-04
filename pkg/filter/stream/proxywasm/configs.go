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

package proxywasm

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
	RootContextID  int32             `json:"root_context_id,omitempty"`
	UserData       map[string]string `json:"-"`
}

func parseFilterConfig(cfg map[string]interface{}) (*filterConfig, error) {
	config := filterConfig{
		UserData:      make(map[string]string),
		RootContextID: 1, // default value is 1
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm][config] fail to marshal filter config, err: %v", err)
		return nil, err
	}

	if err = json.Unmarshal(data, &config); err != nil {
		log.DefaultLogger.Errorf("[proxywasm][config] fail to unmarshal filter config, err: %v", err)
		return nil, err
	}

	if err = checkVmConfig(&config); err != nil {
		log.DefaultLogger.Errorf("[proxywasm][config] fail to check vm config, err: %v", err)
		return nil, err
	}

	if err = parseUserData(data, &config); err != nil {
		log.DefaultLogger.Errorf("[proxywasm][config] fail to parse user data, err: %v", err)
		return nil, err
	}

	return &config, nil
}

func checkVmConfig(config *filterConfig) error {
	if config.FromWasmPlugin != "" {
		config.VmConfig = nil
		config.InstanceNum = 0
	} else {
		if config.VmConfig == nil {
			log.DefaultLogger.Errorf("[proxywasm][config] checkVmConfig fail, nil vm config")
			return errors.New("nil vm config")
		}

		if config.InstanceNum <= 0 {
			config.InstanceNum = runtime.NumCPU()
		}
	}

	return nil
}

func parseUserData(rawConfigBytes []byte, config *filterConfig) error {
	if len(rawConfigBytes) == 0 || config == nil {
		log.DefaultLogger.Errorf("[proxywasm][config] fail to parse user data, invalid param, raw: %v, config: %v",
			string(rawConfigBytes), config)
		return errors.New("invalid param")
	}

	m := make(map[string]interface{})

	if err := json.Unmarshal(rawConfigBytes, &m); err != nil {
		log.DefaultLogger.Errorf("[proxywasm][config] fail to unmarshal user data, err: %v", err)
		return err
	}

	// delete all pairs that value type is not string
	delete(m, "from_wasm_plugin")

	for k, v := range m {
		if _, ok := v.(string); !ok {
			delete(m, k)
		}
	}

	// add into config
	if len(m) > 0 {
		config.UserData = make(map[string]string, len(m))
		for k, v := range m {
			config.UserData[k] = v.(string)
		}
	}

	return nil
}
