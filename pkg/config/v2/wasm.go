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

type WasmPluginConfig struct {
	PluginName  string        `json:"plugin_name,omitempty"`
	VmConfig    *WasmVmConfig `json:"vm_config,omitempty"`
	InstanceNum int           `json:"instance_num,omitempty"`
}

type WasmVmConfig struct {
	Engine string `json:"engine,omitempty"`
	Path   string `json:"path,omitempty"`
	Url    string `json:"url,omitempty"`
	Md5    string `json:"md5,omitempty"`
	Cpu    int    `json:"cpu,omitempty"`
	Mem    int    `json:"mem,omitempty"`
}

func (w WasmPluginConfig) Clone() WasmPluginConfig {
	var vmConfig WasmVmConfig
	if w.VmConfig != nil {
		vmConfig = *w.VmConfig
	}

	return WasmPluginConfig{
		PluginName:  w.PluginName,
		VmConfig:    &vmConfig,
		InstanceNum: w.InstanceNum,
	}
}
