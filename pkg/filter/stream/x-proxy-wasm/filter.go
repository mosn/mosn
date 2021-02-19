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
	"context"
	"sync"
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm"
	"mosn.io/mosn/pkg/wasm/abi"
	"mosn.io/mosn/pkg/wasm/abi/proxywasm_0_1_0"
	"mosn.io/pkg/buffer"
)

type Filter struct {
	proxywasm_0_1_0.DefaultImportsHandler

	ctx context.Context

	factory *FilterConfigFactory

	pluginName string
	plugin     types.WasmPlugin
	instance   types.WasmInstance

	abi     types.ABI
	exports proxywasm_0_1_0.Exports

	rootContextID int32
	contextID     int32

	receiverFilterHandler api.StreamReceiverFilterHandler
	senderFilterHandler   api.StreamSenderFilterHandler

	destroyOnce sync.Once
}

var contextIDGenerator int32

func NewFilter(ctx context.Context, pluginName string, rootContextID int32, factory *FilterConfigFactory) *Filter {
	pluginWrapper := wasm.GetWasmManager().GetWasmPluginWrapperByName(pluginName)
	if pluginWrapper == nil {
		log.DefaultLogger.Errorf("[x-proxy-wasm][filter] NewFilter wasm plugin not exists, plugin name: %v", pluginName)
		return nil
	}

	plugin := pluginWrapper.GetPlugin()
	instance := plugin.GetInstance()

	filter := &Filter{
		ctx:           ctx,
		factory:       factory,
		pluginName:    pluginName,
		plugin:        plugin,
		instance:      instance,
		rootContextID: rootContextID,
		contextID:     atomic.AddInt32(&contextIDGenerator, 1),
	}

	filter.abi = abi.GetABI(instance, proxywasm_0_1_0.ProxyWasmABI_0_1_0)
	if filter.abi == nil {
		log.DefaultLogger.Errorf("[x-proxy-wasm][filter] NewFilter abi not found in instance")
		plugin.ReleaseInstance(instance)
		return nil
	}

	filter.abi.SetImports(filter)

	filter.exports = filter.abi.GetExports().(proxywasm_0_1_0.Exports)
	if filter.exports == nil {
		log.DefaultLogger.Errorf("[x-proxy-wasm][filter] NewFilter fail to get exports part from abi")
		plugin.ReleaseInstance(instance)
		return nil
	}

	filter.instance.Acquire(filter.abi)
	_ = filter.exports.ProxyOnContextCreate(filter.contextID, filter.rootContextID)
	filter.instance.Release()

	return filter
}

func (f *Filter) OnDestroy() {
	f.destroyOnce.Do(func() {
		f.instance.Acquire(f.abi)
		_, _ = f.exports.ProxyOnDone(f.contextID)
		f.instance.Release()

		f.plugin.ReleaseInstance(f.instance)
	})
}

func (f *Filter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.receiverFilterHandler = handler
}

func (f *Filter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.senderFilterHandler = handler
}

func (f *Filter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	f.instance.Acquire(f.abi)
	defer f.instance.Release()

	// do filter
	if buf != nil && buf.Len() > 0 {
		action, err := f.exports.ProxyOnRequestHeaders(f.contextID, 0, 0)
		if err != nil || action != proxywasm_0_1_0.ActionContinue {
			log.DefaultLogger.Errorf("[x-proxy-wasm][filter] OnReceive ProxyOnRequestHeaders err: %v", err)
			return api.StreamFilterStop
		}

		action, err = f.exports.ProxyOnRequestBody(f.contextID, int32(buf.Len()), 1)
		if err != nil || action != proxywasm_0_1_0.ActionContinue {
			log.DefaultLogger.Errorf("[x-proxy-wasm][filter] OnReceive ProxyOnRequestBody err: %v", err)
			return api.StreamFilterStop
		}
	} else {
		action, err := f.exports.ProxyOnRequestHeaders(f.contextID, 0, 1)
		if err != nil || action != proxywasm_0_1_0.ActionContinue {
			log.DefaultLogger.Errorf("[x-proxy-wasm][filter] OnReceive ProxyOnRequestHeaders err: %v", err)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}

func (f *Filter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	f.instance.Acquire(f.abi)
	defer f.instance.Release()

	// do filter
	if buf != nil && buf.Len() > 0 {
		action, err := f.exports.ProxyOnResponseHeaders(f.contextID, 0, 0)
		if err != nil || action != proxywasm_0_1_0.ActionContinue {
			log.DefaultLogger.Errorf("[x-proxy-wasm][filter] OnReceive ProxyOnRequestHeaders err: %v", err)
			return api.StreamFilterStop
		}

		action, err = f.exports.ProxyOnResponseBody(f.contextID, int32(buf.Len()), 1)
		if err != nil || action != proxywasm_0_1_0.ActionContinue {
			log.DefaultLogger.Errorf("[x-proxy-wasm][filter] OnReceive ProxyOnRequestBody err: %v", err)
			return api.StreamFilterStop
		}
	} else {
		action, err := f.exports.ProxyOnResponseHeaders(f.contextID, 0, 1)
		if err != nil || action != proxywasm_0_1_0.ActionContinue {
			log.DefaultLogger.Errorf("[x-proxy-wasm][filter] OnReceive ProxyOnRequestHeaders err: %v", err)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}

func (f *Filter) GetRootContextID() int32 {
	return f.rootContextID
}

func (f *Filter) GetVmConfig() buffer.IoBuffer {
	return f.factory.GetVmConfig()
}

func (f *Filter) GetPluginConfig() buffer.IoBuffer {
	return f.factory.GetPluginConfig()
}

func (f *Filter) GetHttpRequestHeader() api.HeaderMap {
	if f.receiverFilterHandler == nil {
		return nil
	}
	return f.receiverFilterHandler.GetRequestHeaders()
}

func (f *Filter) GetHttpRequestBody() buffer.IoBuffer {
	if f.receiverFilterHandler == nil {
		return nil
	}
	return f.receiverFilterHandler.GetRequestData()
}

func (f *Filter) GetHttpRequestTrailer() api.HeaderMap {
	if f.receiverFilterHandler == nil {
		return nil
	}
	return f.receiverFilterHandler.GetRequestTrailers()
}

func (f *Filter) GetHttpResponseHeader() api.HeaderMap {
	if f.senderFilterHandler == nil {
		return nil
	}
	return f.senderFilterHandler.GetResponseHeaders()
}

func (f *Filter) GetHttpResponseBody() buffer.IoBuffer {
	if f.senderFilterHandler == nil {
		return nil
	}
	return f.senderFilterHandler.GetResponseData()
}

func (f *Filter) GetHttpResponseTrailer() api.HeaderMap {
	if f.senderFilterHandler == nil {
		return nil
	}
	return f.senderFilterHandler.GetResponseTrailers()
}
