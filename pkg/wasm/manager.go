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

// Package wasm implement the wasm framework for MOSN
package wasm

import (
	"errors"
	"reflect"
	"sync"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

var (
	ErrEmptyPluginName   = errors.New("wasm config without plugin name")
	ErrUnexpectedType    = errors.New("unexpected object type in map")
	ErrPluginNotFound    = errors.New("wasm plugin not found")
	ErrSameWasmConfig    = errors.New("update with same wasm config")
	ErrUpdateInstanceNum = errors.New("update instance num return 0")
)

// GetWasmManager returns the global singleton of types.WasmManager.
func GetWasmManager() types.WasmManager {
	return wasmManagerInstance
}

var wasmManagerInstance types.WasmManager = &wasmManagerImpl{}

// implementation of types.WasmManager.
type wasmManagerImpl struct {
	pluginMap sync.Map
}

func (w *wasmManagerImpl) shouldCreateNewPlugin(newConfig v2.WasmPluginConfig, oldConfig v2.WasmPluginConfig) bool {
	if newConfig.VmConfig == nil || oldConfig.VmConfig == nil {
		return false
	}

	if newConfig.VmConfig.Engine != oldConfig.VmConfig.Engine ||
		newConfig.VmConfig.Path != oldConfig.VmConfig.Path ||
		newConfig.VmConfig.Url != oldConfig.VmConfig.Url ||
		newConfig.VmConfig.Md5 == "" ||
		newConfig.VmConfig.Md5 != oldConfig.VmConfig.Md5 {
		return true
	}

	return false
}

func (w *wasmManagerImpl) updateWasm(pluginWrapper types.WasmPluginWrapper, newConfig v2.WasmPluginConfig) error {
	oldConfig := pluginWrapper.GetConfig()
	if reflect.DeepEqual(newConfig, oldConfig) {
		log.DefaultLogger.Infof("[wasm][manager] AddOrUpdateWasm do not update for same config: %v", newConfig)
		return ErrSameWasmConfig
	}

	plugin := pluginWrapper.GetPlugin()

	if w.shouldCreateNewPlugin(newConfig, pluginWrapper.GetConfig()) {
		var err error
		if plugin, err = NewWasmPlugin(newConfig); err != nil {
			log.DefaultLogger.Errorf("[wasm][manager] updateWasm fail to create wasm plugin: %v, err: %v", newConfig.PluginName, err)
			return err
		}
	} else {
		if newConfig.InstanceNum <= 0 {
			newConfig.InstanceNum = plugin.InstanceNum()
		}

		actualNum := plugin.EnsureInstanceNum(newConfig.InstanceNum)
		if actualNum == 0 {
			log.DefaultLogger.Errorf("[wasm][manager] updateWasm fail to update wasm instance num, want num: %v, actual num: %v", newConfig.InstanceNum, actualNum)
			return ErrUpdateInstanceNum
		}
	}

	plugin.SetCpuLimit(newConfig.VmConfig.Cpu)
	plugin.SetMemLimit(newConfig.VmConfig.Mem)

	pluginWrapper.Update(newConfig, plugin)

	log.DefaultLogger.Infof("[wasm][manager] AddOrUpdateWasm update wasm plugin: %v, config: %v", newConfig.PluginName, newConfig)

	return nil
}

func (w *wasmManagerImpl) AddOrUpdateWasm(config v2.WasmPluginConfig) error {
	if config.PluginName == "" {
		log.DefaultLogger.Errorf("[wasm][manager] AddOrUpdateWasm empty plugin name")
		return ErrEmptyPluginName
	}

	// plugin already exists
	if v, ok := w.pluginMap.Load(config.PluginName); ok {
		pluginWrapper, ok := v.(*pluginWrapper)
		if !ok {
			log.DefaultLogger.Errorf("[wasm][manager] AddOrUpdateWasm unexpected type in map")
			return ErrUnexpectedType
		}

		return w.updateWasm(pluginWrapper, config)
	}

	// add new wasm plugin
	plugin, err := NewWasmPlugin(config)
	if err != nil {
		log.DefaultLogger.Errorf("[wasm][manager] AddOrUpdateWasm fail to create wasm plugin: %v, err: %v", config.PluginName, err)
		return err
	}

	pw := &pluginWrapper{
		plugin: plugin,
		config: config,
	}

	w.pluginMap.LoadOrStore(config.PluginName, pw)

	log.DefaultLogger.Infof("[wasm][manager] AddOrUpdateWasm add new wasm plugin: %v", config.PluginName)

	return nil
}

func (w *wasmManagerImpl) GetWasmPluginWrapperByName(pluginName string) types.WasmPluginWrapper {
	if pluginName == "" {
		return nil
	}

	if v, ok := w.pluginMap.Load(pluginName); ok {
		pw, ok := v.(*pluginWrapper)
		if !ok {
			log.DefaultLogger.Errorf("[wasm][manager] GetWasmPluginWrapperByName unexpected object type in map")
			return nil
		}

		return pw
	}

	log.DefaultLogger.Errorf("[wasm][manager] GetWasmPluginWrapperByName not found in map, plugin name: %v", pluginName)

	return nil
}

func (w *wasmManagerImpl) UninstallWasmPluginByName(pluginName string) error {
	v, ok := w.pluginMap.Load(pluginName)
	if !ok {
		log.DefaultLogger.Errorf("[wasm][manager] UninstallWasmPluginByName plugin not found, name: %v", pluginName)
		return ErrPluginNotFound
	}

	w.pluginMap.Delete(pluginName)

	pw := v.(*pluginWrapper)
	pw.GetPlugin().Clear()

	log.DefaultLogger.Infof("[wasm][manager] UninstallWasmPluginByName uninstall wasm plugin: %v", pluginName)

	return nil
}
