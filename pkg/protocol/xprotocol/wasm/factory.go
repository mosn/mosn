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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm"
	"mosn.io/mosn/pkg/wasm/abi"
	"mosn.io/mosn/pkg/wasm/abi/ext/xproxywasm020"
	"mosn.io/mosn/pkg/wasm/abi/proxywasm010"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
)

func GetProxyProtocolManager() ProxyProtocolManager {
	return proxyProtocolMng
}

type ProxyProtocolManager interface {

	// AddOrUpdateProtocolConfig map the key to streamFilter chain wrapper.
	AddOrUpdateProtocolConfig(config map[string]interface{}) error

	// GetWasmProtocolFactory return StreamFilterFactory indexed by key.
	GetWasmProtocolFactory(key string) ProxyProtocolWrapper
}

var proxyProtocolMng = &wasmProtocolManager{factory: make(map[string]ProxyProtocolWrapper)}

type wasmProtocolManager struct {
	factory map[string]ProxyProtocolWrapper
}

func (m *wasmProtocolManager) AddOrUpdateProtocolConfig(config map[string]interface{}) error {
	factory, err := createProxyWasmProtocolFactory(config)
	if err != nil {
		log.DefaultLogger.Errorf("failed to create wasm proxy protocol wrapper, err %v", err)
		return err
	}
	if factory != nil {
		m.factory[factory.Name()] = factory
	}
	return nil
}

func (m *wasmProtocolManager) GetWasmProtocolFactory(key string) ProxyProtocolWrapper {
	return m.factory[key]
}

type ProxyProtocolWrapper interface {
	Name() string
}

type protocolWrapper struct {
	proxywasm010.DefaultImportsHandler
	pluginName    string
	config        *ProtocolConfig        // plugin config
	conf          map[string]interface{} // raw config map
	pluginWrapper types.WasmPluginWrapper
}

func (f *protocolWrapper) Name() string {
	return f.config.SubProtocol
}

func createProxyWasmProtocolFactory(conf map[string]interface{}) (ProxyProtocolWrapper, error) {
	config, err := ParseProtocolConfig(conf)
	if err != nil {
		log.DefaultLogger.Errorf("[wasm][protocol] CreateProxyWasmProtocolFactory fail to parse wrapper, err: %v", err)
		return nil, err
	}

	if config == nil {
		return nil, nil
	}

	// not wasm extension
	if config.FromWasmPlugin == "" && config.VmConfig == nil {
		return nil, nil
	}

	var pluginName string
	if config.FromWasmPlugin != "" {
		pluginName = config.FromWasmPlugin
	} else {
		pluginName = utils.GenerateUUID()

		v2Config := v2.WasmPluginConfig{
			PluginName:  pluginName,
			VmConfig:    config.VmConfig,
			InstanceNum: config.InstanceNum,
		}

		err = wasm.GetWasmManager().AddOrUpdateWasm(v2Config)
		if err != nil {
			log.DefaultLogger.Errorf("[wasm][protocol] CreateProxyWasmProtocolFactory fail to add plugin, err: %v", err)
			return nil, err
		}
	}

	pw := wasm.GetWasmManager().GetWasmPluginWrapperByName(pluginName)
	if pw == nil {
		return nil, errors.New(fmt.Sprintf("plugin '%s' not found", pluginName))
	}

	if config.FromWasmPlugin == "" {
		config.VmConfig = pw.GetConfig().VmConfig
	}

	wrapper := &protocolWrapper{
		pluginName:    pluginName,
		config:        config,
		conf:          conf,
		pluginWrapper: pw,
	}

	name := types.ProtocolName(config.SubProtocol)
	if ok := protocol.ProtocolRegistered(name); !ok {
		// first time plugin init, register proxy protocol.
		// because the listener contains the types egress and ingress,
		// the plug-in should be fired only once.
		// TODO: support factory, mapping and matcher
		codec := &xCodec{
			proto: NewWasmRpcProtocol(pw, wrapper),
		}
		_ = xprotocol.RegisterXProtocolCodec(codec)
		// invoke plugin lifecycle pipeline
		pw.RegisterPluginHandler(wrapper)
		// todo detect plugin feature (ping-pong, worker pool?)
	}

	return wrapper, nil
}

func (f *protocolWrapper) OnConfigUpdate(config v2.WasmPluginConfig) {
	f.config.InstanceNum = config.InstanceNum
	f.config.VmConfig = config.VmConfig
}

func (f *protocolWrapper) OnPluginStart(plugin types.WasmPlugin) {
	plugin.Exec(func(instance types.WasmInstance) bool {

		abiVersion := abi.GetABI(instance, xproxywasm020.AbiV2)
		abiVersion.SetABIImports(f)
		exports := abiVersion.(xproxywasm020.Exports)

		if abiVersion == nil {
			log.DefaultLogger.Errorf("[wasm][protocol] NewProtocol abi version not found")
			return true
		}

		instance.Lock(abiVersion)
		defer instance.Unlock()

		err := exports.ProxyOnContextCreate(f.config.RootContextID, 0)
		if err != nil {
			log.DefaultLogger.Errorf("[wasm] failed to create plugin % root context %d , err: %v", plugin.PluginName(), f.config.RootContextID, err)
		}
		_, err = exports.ProxyOnVmStart(f.config.RootContextID, int32(f.GetVmConfig().Len()))
		if err != nil {
			log.DefaultLogger.Errorf("[wasm] failed to start plugin %, root contextId %d , err: %v", plugin.PluginName(), f.config.RootContextID, err)
		}
		_, err = exports.ProxyOnConfigure(f.config.RootContextID, int32(f.GetPluginConfig().Len()))
		if err != nil {
			log.DefaultLogger.Errorf("[wasm] failed to configure plugin %, root contextId %d , err: %v", plugin.PluginName(), f.config.RootContextID, err)
		}

		return true
	})
}

func (f *protocolWrapper) OnPluginDestroy(_ types.WasmPlugin) {
	return
}

func (f *protocolWrapper) GetVmConfig() common.IoBuffer {
	if f.config.VmConfig != nil {
		configs := make(map[string]string, 8)
		typeOf := reflect.TypeOf(*f.config.VmConfig)
		values := reflect.ValueOf(*f.config.VmConfig)
		for i := 0; i < typeOf.NumField(); i++ {
			field := values.Field(i)
			switch field.Kind() {
			case reflect.String:
				configs[typeOf.Field(i).Name] = field.String()
			}
		}

		vmBuffer := buffer.NewIoBufferBytes(EncodeMap(configs))
		return vmBuffer
	}
	return buffer.NewIoBufferEOF()
}

func (f *protocolWrapper) GetPluginConfig() common.IoBuffer {
	configs := make(map[string]string, 8)
	for key, v := range f.conf {
		if value, ok := v.(string); ok {
			configs[key] = value
		}
	}
	pluginBuffer := buffer.NewIoBufferBytes(EncodeMap(configs))
	maps := DecodeMap(pluginBuffer.Bytes())
	_ = maps
	return pluginBuffer
}

func ParseProtocolConfig(cfg map[string]interface{}) (*ProtocolConfig, error) {
	var config ProtocolConfig

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.FromWasmPlugin != "" {
		config.VmConfig = nil
	}

	switch config.PoolMode {
	case "ping-pong":
		config.poolMode = api.PingPong
	case "tcp":
		config.poolMode = api.TCP
	default:
		config.poolMode = api.Multiplex
	}

	return &config, nil
}
