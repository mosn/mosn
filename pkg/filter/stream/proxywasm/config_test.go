//go:build wasmer
// +build wasmer

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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXProxyWasmConfigFromGlobalPlugin(t *testing.T) {
	configMap := map[string]interface{}{
		"from_wasm_plugin": "global_plugin",
		"instance_num":     2,
		"user_config1":     "user_value1",
		"user_config2":     "user_value2",
	}

	config, err := parseFilterConfig(configMap)
	assert.Nil(t, err)
	assert.Nil(t, config.VmConfig)
	assert.Equal(t, config.InstanceNum, 0)
	assert.Equal(t, len(config.UserData), 2)
}

func TestXProxyWasmConfigWithoutUserData(t *testing.T) {
	configMap := map[string]interface{}{
		"from_wasm_plugin": "global_plugin",
		"instance_num":     2,
	}

	config, err := parseFilterConfig(configMap)
	assert.Nil(t, err)
	assert.Nil(t, config.VmConfig)
	assert.Equal(t, config.InstanceNum, 0)
	assert.Equal(t, len(config.UserData), 0)
}

func TestXProxyWasmConfigWithVM(t *testing.T) {
	configMap := map[string]interface{}{
		"vm_config": map[string]interface{}{
			"engine": "wasmer",
			"path":   "path",
			"cpu":    100,
		},
		"instance_num": 2,
		"user_config1": "user_value1",
		"user_config2": "user_value2",
		"user_config3": "user_value3",
		"user_config4": "user_value4",
	}

	config, err := parseFilterConfig(configMap)
	assert.Nil(t, err)
	assert.Equal(t, config.FromWasmPlugin, "")
	assert.NotNil(t, config.VmConfig)
	assert.Equal(t, config.InstanceNum, 2)
	assert.Equal(t, len(config.UserData), 4)
}
