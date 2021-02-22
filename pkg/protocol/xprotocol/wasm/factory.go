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
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm"
	"mosn.io/mosn/pkg/wasm/abi"
	"mosn.io/mosn/pkg/wasm/abi/proxywasm_0_1_0"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
	"reflect"
)

func GetProxyProtocolManager() ProxyProtocolManager {
	return proxyProtocolMng
}

type ProxyProtocolManager interface {

	// AddOrUpdateProtocolConfig map the key to streamFilter chain config.
	AddOrUpdateProtocolConfig(config map[string]interface{}) error

	// GetWasmProtocolFactory return StreamFilterFactory indexed by key.
	GetWasmProtocolFactory(key string) ProxyProtocolFactory
}

var proxyProtocolMng = &wasmProtocolManager{factory: make(map[string]ProxyProtocolFactory)}

type wasmProtocolManager struct {
	factory map[string]ProxyProtocolFactory
}

func (m *wasmProtocolManager) AddOrUpdateProtocolConfig(config map[string]interface{}) error {
	factory, err := createProxyWasmProtocolFactory(config)
	if err != nil {
		log.DefaultLogger.Errorf("failed to create wasm proxy protocol factory, err %v", err)
		return err
	}
	if factory != nil {
		m.factory[factory.Name()] = factory
	}
	return nil
}

func (m *wasmProtocolManager) GetWasmProtocolFactory(key string) ProxyProtocolFactory {
	return m.factory[key]
}

type ProxyProtocolFactory interface {
	Name() string
}

type protocolFactory struct {
	proxywasm_0_1_0.DefaultInstanceCallback
	pluginName    string
	config        *ProtocolConfig        // plugin config
	conf          map[string]interface{} // raw config map
	pluginWrapper types.WasmPluginWrapper
}

func (f *protocolFactory) Name() string {
	return f.config.ExtendConfig.SubProtocol
}

func createProxyWasmProtocolFactory(conf map[string]interface{}) (ProxyProtocolFactory, error) {
	config, err := ParseProtocolConfig(conf)
	if err != nil {
		log.DefaultLogger.Errorf("[wasm][protocol] CreateProxyWasmProtocolFactory fail to parse config, err: %v", err)
		return nil, err
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

	factory := &protocolFactory{
		pluginName:    pluginName,
		config:        config,
		conf:          conf,
		pluginWrapper: pw,
	}

	name := types.ProtocolName(config.ExtendConfig.SubProtocol)
	p := xprotocol.GetProtocol(name)
	if p == nil {
		// first time plugin init, register proxy protocol.
		// because the listener contains the types egress and ingress,
		// the plug-in should be fired only once.
		p = NewWasmRpcProtocol(config, pw.GetPlugin().GetInstance())
		_ = xprotocol.RegisterProtocol(name, p)
		// invoke plugin lifecycle pipeline
		pw.RegisterPluginHandler(factory)
		// todo detect plugin feature (ping-pong, worker pool?)
	}

	return factory, nil
}

func (f *protocolFactory) OnConfigUpdate(config v2.WasmPluginConfig) {
	f.config.InstanceNum = config.InstanceNum
	f.config.VmConfig = config.VmConfig
}

func (f *protocolFactory) OnPluginStart(plugin types.WasmPlugin) {
	abiVersion := abi.GetABI("proxy_abi_version_0_2_0")
	if abiVersion == nil {
		log.DefaultLogger.Errorf("[wasm][protocol] NewProtocol abi version not found")
		return
	}

	plugin.Exec(func(instanceWrapper types.WasmInstanceWrapper) bool {
		instanceWrapper.Acquire(f)
		defer instanceWrapper.Release()

		abiVersion.SetInstance(instanceWrapper)
		exports := abiVersion.(Exports)

		_ = exports.ProxyOnContextCreate(f.config.RootContextID, 0)
		//_, _ = exports.ProxyOnVmStart(f.config.RootContextID, int32(f.GetVmConfig().Len()))
		_, _ = exports.ProxyOnConfigure(f.config.RootContextID, int32(f.GetPluginConfig().Len()))

		return true
	})
}

func (f *protocolFactory) OnPluginDestroy(_ types.WasmPlugin) {
	return
}

func (f *protocolFactory) GetVmConfig() buffer.IoBuffer {
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

func (f *protocolFactory) GetPluginConfig() buffer.IoBuffer {
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
		config.poolMode = types.PingPong
	case "tcp":
		config.poolMode = types.TCP
	default:
		config.poolMode = types.Multiplex
	}

	return &config, nil
}
