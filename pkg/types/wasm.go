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

package types

import (
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
)

//
//	Manager
//

// WasmManager managers all wasm configs
type WasmManager interface {
	// AddOrUpdateWasm add or update wasm plugin config
	AddOrUpdateWasm(wasmConfig v2.WasmPluginConfig) error

	// GetWasmPluginWrapperByName returns wasm plugin by name
	GetWasmPluginWrapperByName(pluginName string) WasmPluginWrapper

	// UninstallWasmPluginByName remove wasm plugin by name
	UninstallWasmPluginByName(pluginName string) error
}

// WasmPluginWrapper wraps wasm plugin with its config and plugin handler
type WasmPluginWrapper interface {
	// GetPlugin returns the wasm plugin
	GetPlugin() WasmPlugin

	// GetConfig returns the config of wasm plugin
	GetConfig() v2.WasmPluginConfig

	// RegisterPluginHandler registers a plugin handler for the wasm plugin
	RegisterPluginHandler(pluginHandler WasmPluginHandler)

	// Update updates the plugin
	Update(config v2.WasmPluginConfig, plugin WasmPlugin)
}

// WasmPluginHandler provides callbacks to manager the life cycle of the wasm plugin
type WasmPluginHandler interface {
	// OnConfigUpdate got called when updating the config of the wasm plugin
	OnConfigUpdate(config v2.WasmPluginConfig)

	// OnPluginStart got called when starting the wasm plugin
	OnPluginStart(plugin WasmPlugin)

	// OnPluginDestroy got called when destroying the wasm plugin
	OnPluginDestroy(plugin WasmPlugin)
}

// WasmPlugin manages the collection of wasm plugin instances
type WasmPlugin interface {
	// PluginName returns the name of wasm plugin
	PluginName() string

	// GetPluginConfig returns the config of wasm plugin
	GetPluginConfig() v2.WasmPluginConfig

	// GetVmConfig returns the vm config of wasm plugin
	GetVmConfig() v2.WasmVmConfig

	// EnsureInstanceNum tries to expand/shrink the num of instance to 'num'
	// and returns the actual instance num
	EnsureInstanceNum(num int) int

	// InstanceNum returns the current number of wasm instance
	InstanceNum() int

	// GetInstance returns one plugin instance of the plugin
	GetInstance() WasmInstance

	// ReleaseInstance releases the instance to the plugin
	ReleaseInstance(instance WasmInstance)

	// Exec execute the f for each instance
	Exec(f func(instance WasmInstance) bool)

	// SetCpuLimit set cpu limit of the plugin, not supported currently
	SetCpuLimit(cpu int)

	// SetMemLimit set cpu limit of the plugin, not supported currently
	SetMemLimit(mem int)

	// Clear got called when the plugin is destroyed
	Clear()
}

//
// VM
//

type WasmVM = common.WasmVM

type WasmModule = common.WasmModule

type WasmInstance = common.WasmInstance

type WasmFunction = common.WasmFunction

//
//	ABI
//

// ABI represents the abi between the host and wasm, which consists of three parts: exports, imports and life-cycle handler
// *exports* represents the exported elements of the wasm module, i.e., the abilities provided by wasm and exposed to host
// *imports* represents the imported elements of the wasm module, i.e., the dependencies that required by wasm
// *life-cycle handler* manages the life-cycle of an abi
type ABI interface {
	// Name returns the name of ABI
	Name() string

	// GetABIImports gets the imports part of the abi
	GetABIImports() interface{}

	// SetImports sets the import part of the abi
	SetABIImports(imports interface{})

	// GetExports returns the export part of the abi
	GetABIExports() interface{}

	ABIHandler
}

type ABIHandler interface {
	// life-cycle: OnInstanceCreate got called when instantiating the wasm instance
	OnInstanceCreate(instance WasmInstance)

	// life-cycle: OnInstanceStart got called when starting the wasm instance
	OnInstanceStart(instance WasmInstance)

	// life-cycle: OnInstanceDestroy got called when destroying the wasm instance
	OnInstanceDestroy(instance WasmInstance)
}
