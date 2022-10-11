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
	"context"
	"errors"
	"fmt"
	"reflect"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm"
	wasmABI "mosn.io/mosn/pkg/wasm/abi"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
	v1Host "mosn.io/proxy-wasm-go-host/proxywasm/v1"
	v2Host "mosn.io/proxy-wasm-go-host/proxywasm/v2"
)

const ProxyWasm = "proxywasm"

func init() {
	api.RegisterStream(ProxyWasm, createProxyWasmFilterFactory)
}

type FilterConfigFactory struct {
	pluginName    string
	config        *filterConfig
	rootContextID int32

	vmConfigBytes     buffer.IoBuffer
	pluginConfigBytes buffer.IoBuffer
}

func createProxyWasmFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	config, err := parseFilterConfig(conf)
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm][factory] createProxyWasmFilterFactory fail to parse config, err: %v", err)
		return nil, err
	}

	var pluginName string

	if config.FromWasmPlugin == "" {
		pluginName = utils.GenerateUUID()

		v2Config := v2.WasmPluginConfig{
			PluginName:  pluginName,
			VmConfig:    config.VmConfig,
			InstanceNum: config.InstanceNum,
		}

		err = wasm.GetWasmManager().AddOrUpdateWasm(v2Config)
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm][factory] createProxyWasmFilterFactory fail to add plugin, err: %v", err)
			return nil, err
		}
	} else {
		pluginName = config.FromWasmPlugin
	}

	pw := wasm.GetWasmManager().GetWasmPluginWrapperByName(pluginName)
	if pw == nil {
		return nil, errors.New("plugin not found")
	}

	config.VmConfig = pw.GetConfig().VmConfig

	factory := &FilterConfigFactory{
		pluginName:    pluginName,
		config:        config,
		rootContextID: config.RootContextID,
	}

	pw.RegisterPluginHandler(factory)

	return factory, nil
}

func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewFilter(context, f.pluginName, f.config.RootContextID, f)
	if filter == nil {
		return
	}

	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
	callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}

func (f *FilterConfigFactory) GetVmConfig() common.IoBuffer {
	if f.vmConfigBytes != nil {
		return f.vmConfigBytes
	}

	vmConfig := *f.config.VmConfig
	typeOf := reflect.TypeOf(vmConfig)
	valueOf := reflect.ValueOf(&vmConfig).Elem()

	if typeOf.Kind() != reflect.Struct || typeOf.NumField() == 0 {
		return nil
	}

	m := make(map[string]string)
	for i := 0; i < typeOf.NumField(); i++ {
		m[typeOf.Field(i).Name] = fmt.Sprintf("%v", valueOf.Field(i).Interface())
	}

	b := common.EncodeMap(m)
	if b == nil {
		return nil
	}

	f.vmConfigBytes = buffer.NewIoBufferBytes(b)

	return f.vmConfigBytes
}

func (f *FilterConfigFactory) GetPluginConfig() common.IoBuffer {
	if f.pluginConfigBytes != nil {
		return f.pluginConfigBytes
	}

	b := common.EncodeMap(f.config.UserData)
	if b == nil {
		return nil
	}

	f.pluginConfigBytes = buffer.NewIoBufferBytes(b)

	return f.pluginConfigBytes
}

func (f *FilterConfigFactory) OnConfigUpdate(config v2.WasmPluginConfig) {
	f.config.InstanceNum = config.InstanceNum
	f.config.VmConfig = config.VmConfig
}

func (f *FilterConfigFactory) OnPluginStart(plugin types.WasmPlugin) {
	plugin.Exec(func(instance types.WasmInstance) bool {
		abi := wasmABI.GetABIList(instance)[0]
		var exports *exportAdapter
		if abi.Name() == v1Host.ProxyWasmABI_0_1_0 {
			// v1
			imports := &v1Imports{factory: f}
			imports.DefaultImportsHandler.Instance = instance
			abi.SetABIImports(imports)
			exports = &exportAdapter{v1: abi.GetABIExports().(v1Host.Exports)}
		} else if abi.Name() == v2Host.ProxyWasmABI_0_2_0 {
			// v2
			imports := &v2Imports{factory: f}
			imports.DefaultImportsHandler.Instance = instance
			abi.SetABIImports(imports)
			exports = &exportAdapter{v2: abi.GetABIExports().(v2Host.Exports)}
		} else {
			log.DefaultLogger.Errorf("[proxywasm][factory] unknown abi list: %v", abi)
			return false
		}

		instance.Lock(abi)
		defer instance.Unlock()

		err := exports.ProxyOnContextCreate(f.config.RootContextID, 0)
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm][factory] OnPluginStart fail to create root context id, err: %v", err)
			return true
		}

		vmConfigSize := 0
		if vmConfigBytes := f.GetVmConfig(); vmConfigBytes != nil {
			vmConfigSize = vmConfigBytes.Len()
		}

		_, err = exports.ProxyOnVmStart(f.config.RootContextID, int32(vmConfigSize))
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm][factory] OnPluginStart fail to create root context id, err: %v", err)
			return true
		}

		pluginConfigSize := 0
		if pluginConfigBytes := f.GetPluginConfig(); pluginConfigBytes != nil {
			pluginConfigSize = pluginConfigBytes.Len()
		}

		_, err = exports.ProxyOnConfigure(f.config.RootContextID, int32(pluginConfigSize))
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm][factory] OnPluginStart fail to create root context id, err: %v", err)
			return true
		}

		return true
	})
}

func (f *FilterConfigFactory) OnPluginDestroy(plugin types.WasmPlugin) {}
