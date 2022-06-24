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

package wasm

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

var (
	ErrEngineNotFound     = errors.New("fail to get wasm engine")
	ErrWasmBytesLoad      = errors.New("fail to load wasm bytes")
	ErrWasmBytesIncorrect = errors.New("incorrect hash of wasm bytes")
	ErrInstanceCreate     = errors.New("fail to create wasm instance")
	ErrModuleCreate       = errors.New("fail to create wasm module")
)

type pluginWrapper struct {
	mu             sync.RWMutex
	plugin         types.WasmPlugin
	config         v2.WasmPluginConfig
	pluginHandlers []types.WasmPluginHandler
}

func (w *pluginWrapper) RegisterPluginHandler(pluginHandler types.WasmPluginHandler) {
	if pluginHandler == nil {
		return
	}

	w.mu.Lock()
	w.pluginHandlers = append(w.pluginHandlers, pluginHandler)
	w.mu.Unlock()

	pluginHandler.OnConfigUpdate(w.config)
	pluginHandler.OnPluginStart(w.plugin)
}

func (w *pluginWrapper) GetPlugin() types.WasmPlugin {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.plugin
}

func (w *pluginWrapper) GetConfig() v2.WasmPluginConfig {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.config
}

func (w *pluginWrapper) Update(config v2.WasmPluginConfig, plugin types.WasmPlugin) {
	if config.PluginName == "" || config.PluginName != w.GetConfig().PluginName {
		return
	}

	// update config
	for _, handler := range w.pluginHandlers {
		handler.OnConfigUpdate(config)
	}

	w.mu.Lock()
	w.config = config
	w.mu.Unlock()

	// update plugin
	if plugin == nil {
		return
	}

	// check same plugin
	oldPlugin := w.GetPlugin()
	if plugin == oldPlugin {
		return
	}

	// do update plugin
	for _, handler := range w.pluginHandlers {
		handler.OnPluginStart(plugin)
	}

	w.mu.Lock()
	w.plugin = plugin
	w.mu.Unlock()

	for _, handler := range w.pluginHandlers {
		handler.OnPluginDestroy(oldPlugin)
	}

	oldPlugin.Clear()
}

type wasmPluginImpl struct {
	config v2.WasmPluginConfig

	lock sync.RWMutex

	instanceNum    int32
	instances      []types.WasmInstance
	instancesIndex int32

	occupy int32

	vm        types.WasmVM
	wasmBytes []byte
	module    types.WasmModule
}

func NewWasmPlugin(wasmConfig v2.WasmPluginConfig) (types.WasmPlugin, error) {
	// check instance num
	instanceNum := wasmConfig.InstanceNum
	if instanceNum <= 0 {
		instanceNum = runtime.NumCPU()
	}

	wasmConfig.InstanceNum = instanceNum

	// get wasm engine
	vm := GetWasmEngine(wasmConfig.VmConfig.Engine)
	if vm == nil {
		log.DefaultLogger.Errorf("[wasm][plugin] NewWasmPlugin fail to get wasm engine: %v", wasmConfig.VmConfig.Engine)
		return nil, ErrEngineNotFound
	}

	// load wasm bytes
	var wasmBytes []byte
	if wasmConfig.VmConfig.Path != "" {
		wasmBytes = loadWasmBytesFromPath(wasmConfig.VmConfig.Path)
	} else {
		wasmBytes = loadWasmBytesFromUrl(wasmConfig.VmConfig.Url)
	}

	if len(wasmBytes) == 0 {
		log.DefaultLogger.Errorf("[wasm][plugin] NewWasmPlugin fail to load wasm bytes, config: %v", wasmConfig)
		return nil, ErrWasmBytesLoad
	}

	md5Bytes := md5.Sum(wasmBytes)
	newMd5 := hex.EncodeToString(md5Bytes[:])
	if wasmConfig.VmConfig.Md5 == "" {
		wasmConfig.VmConfig.Md5 = newMd5
	} else if newMd5 != wasmConfig.VmConfig.Md5 {
		log.DefaultLogger.Errorf("[wasm][plugin] NewWasmPlugin the hash(MD5) of wasm bytes is incorrect, config: %v, real hash: %s",
			wasmConfig, newMd5)
		return nil, ErrWasmBytesIncorrect
	}

	// create wasm module
	module := vm.NewModule(wasmBytes)
	if module == nil {
		log.DefaultLogger.Errorf("[wasm][plugin] NewWasmPlugin fail to create module, config: %v", wasmConfig)
		return nil, ErrModuleCreate
	}

	plugin := &wasmPluginImpl{
		config:    wasmConfig,
		vm:        vm,
		wasmBytes: wasmBytes,
		module:    module,
	}

	plugin.SetCpuLimit(wasmConfig.VmConfig.Cpu)
	plugin.SetMemLimit(wasmConfig.VmConfig.Mem)

	// ensure instance num
	actual := plugin.EnsureInstanceNum(wasmConfig.InstanceNum)
	if actual == 0 {
		log.DefaultLogger.Errorf("[wasm][plugin] NewWasmPlugin fail to ensure instance num, want: %v got 0", instanceNum)
		return nil, ErrInstanceCreate
	}

	return plugin, nil
}

// EnsureInstanceNum try to expand/shrink the num of instance to 'num'
// and return the actual instance num.
func (w *wasmPluginImpl) EnsureInstanceNum(num int) int {
	if num <= 0 || num == w.InstanceNum() {
		return w.InstanceNum()
	}

	if num < w.InstanceNum() {
		w.lock.Lock()

		for i := num; i < len(w.instances); i++ {
			w.instances[i].Stop()
			w.instances[i] = nil
		}

		w.instances = w.instances[:num]
		atomic.StoreInt32(&w.instanceNum, int32(num))

		w.lock.Unlock()
	} else {
		newInstance := make([]types.WasmInstance, 0)
		numToCreate := num - w.InstanceNum()

		for i := 0; i < numToCreate; i++ {
			instance := w.module.NewInstance()
			if instance == nil {
				log.DefaultLogger.Errorf("[wasm][plugin] EnsureInstanceNum fail to create instance, i: %v", i)
				continue
			}

			err := instance.Start()
			if err != nil {
				log.DefaultLogger.Errorf("[wasm][plugin] EnsureInstanceNum fail to start instance, i: %v, err: %v", i, err)
				continue
			}

			newInstance = append(newInstance, instance)
		}

		w.lock.Lock()

		w.instances = append(w.instances, newInstance...)
		atomic.AddInt32(&w.instanceNum, int32(len(newInstance)))

		w.lock.Unlock()
	}

	return w.InstanceNum()
}

func (w *wasmPluginImpl) InstanceNum() int {
	return int(atomic.LoadInt32(&w.instanceNum))
}

func (w *wasmPluginImpl) PluginName() string {
	return w.config.PluginName
}

func (w *wasmPluginImpl) Clear() {
	// do nothing
	log.DefaultLogger.Infof("[wasm][plugin] Clear wasm plugin, config: %v, instanceNum: %v", w.config, w.instanceNum)
}

// SetCpuLimit set cpu limit of the plugin, no-op.
func (w *wasmPluginImpl) SetCpuLimit(cpu int) {}

// SetMemLimit set memory limit of the plugin, no-op.
func (w *wasmPluginImpl) SetMemLimit(mem int) {}

// Exec execute the f for each instance.
func (w *wasmPluginImpl) Exec(f func(instance types.WasmInstance) bool) {
	w.lock.RLock()
	defer w.lock.RUnlock()

	for _, iw := range w.instances {
		if !f(iw) {
			break
		}
	}
}

func (w *wasmPluginImpl) GetPluginConfig() v2.WasmPluginConfig {
	return w.config
}

func (w *wasmPluginImpl) GetVmConfig() v2.WasmVmConfig {
	return *w.config.VmConfig
}

func (w *wasmPluginImpl) GetInstance() types.WasmInstance {
	w.lock.RLock()
	defer w.lock.RUnlock()

	for i := 0; i < len(w.instances); i++ {
		idx := int(atomic.LoadInt32(&w.instancesIndex)) % len(w.instances)
		atomic.AddInt32(&w.instancesIndex, 1)

		instance := w.instances[idx]
		if !instance.Acquire() {
			continue
		}

		atomic.AddInt32(&w.occupy, 1)
		return instance
	}
	log.DefaultLogger.Errorf("[wasm][plugin] GetInstance fail to get available instance, instance num: %v", len(w.instances))

	return nil
}

func (w *wasmPluginImpl) ReleaseInstance(instance types.WasmInstance) {
	instance.Release()
	atomic.AddInt32(&w.occupy, -1)
}

// DefaultWasmPluginHandler is the default implementation of types.WasmPluginHandler.
type DefaultWasmPluginHandler struct{}

func (d *DefaultWasmPluginHandler) OnConfigUpdate(config v2.WasmPluginConfig) {}

func (d *DefaultWasmPluginHandler) OnPluginStart(plugin types.WasmPlugin) {}

func (d *DefaultWasmPluginHandler) OnPluginDestroy(plugin types.WasmPlugin) {}
