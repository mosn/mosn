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
	"sync"
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm"
	"mosn.io/mosn/pkg/wasm/abi"
	"mosn.io/mosn/pkg/wasm/abi/proxywasm010"
	"mosn.io/pkg/buffer"
	"mosn.io/proxy-wasm-go-host/common"
	"mosn.io/proxy-wasm-go-host/proxywasm"
)

type Filter struct {
	proxywasm010.DefaultImportsHandler

	ctx context.Context

	factory *FilterConfigFactory

	pluginName string
	plugin     types.WasmPlugin
	instance   types.WasmInstance
	abi        types.ABI
	exports    proxywasm.Exports

	rootContextID int32
	contextID     int32

	receiverFilterHandler api.StreamReceiverFilterHandler
	senderFilterHandler   api.StreamSenderFilterHandler

	destroyOnce sync.Once
}

var contextIDGenerator int32

func newContextID(rootContextID int32) int32 {
	for {
		id := atomic.AddInt32(&contextIDGenerator, 1)
		if id != rootContextID {
			return id
		}
	}
}

func NewFilter(ctx context.Context, pluginName string, rootContextID int32, factory *FilterConfigFactory) *Filter {
	pluginWrapper := wasm.GetWasmManager().GetWasmPluginWrapperByName(pluginName)
	if pluginWrapper == nil {
		log.DefaultLogger.Errorf("[proxywasm][filter] NewFilter wasm plugin not exists, plugin name: %v", pluginName)
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
		contextID:     newContextID(rootContextID),
	}

	filter.DefaultImportsHandler.Instance = instance

	filter.abi = abi.GetABI(instance, proxywasm.ProxyWasmABI_0_1_0)
	if filter.abi == nil {
		log.DefaultLogger.Errorf("[proxywasm][filter] NewFilter abi not found in instance")
		plugin.ReleaseInstance(instance)

		return nil
	}

	filter.abi.SetABIImports(filter)

	filter.exports = filter.abi.GetABIExports().(proxywasm.Exports)
	if filter.exports == nil {
		log.DefaultLogger.Errorf("[proxywasm][filter] NewFilter fail to get exports part from abi")
		plugin.ReleaseInstance(instance)

		return nil
	}

	filter.instance.Lock(filter.abi)
	defer filter.instance.Unlock()

	err := filter.exports.ProxyOnContextCreate(filter.contextID, filter.rootContextID)
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm][filter] NewFilter fail to create context id: %v, rootContextID: %v, err: %v",
			filter.contextID, filter.rootContextID, err)
		return nil
	}

	return filter
}

func (f *Filter) OnDestroy() {
	f.destroyOnce.Do(func() {
		f.instance.Lock(f.abi)

		_, err := f.exports.ProxyOnDone(f.contextID)
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm][filter] OnDestroy fail to call ProxyOnDone, err: %v", err)
		}

		err = f.exports.ProxyOnDelete(f.contextID)
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm][filter] OnDestroy fail to call ProxyOnDelete, err: %v", err)
		}

		f.instance.Unlock()
		f.plugin.ReleaseInstance(f.instance)
	})
}

func (f *Filter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.receiverFilterHandler = handler
}

func (f *Filter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.senderFilterHandler = handler
}

func headerMapSize(headers api.HeaderMap) int {
	size := 0

	if headers != nil {
		headers.Range(func(key, value string) bool {
			size++
			return true
		})
	}

	return size
}

func (f *Filter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	f.instance.Lock(f.abi)
	defer f.instance.Unlock()

	endOfStream := 1
	if (buf != nil && buf.Len() > 0) || trailers != nil {
		endOfStream = 0
	}

	action, err := f.exports.ProxyOnRequestHeaders(f.contextID, int32(headerMapSize(headers)), int32(endOfStream))
	if err != nil || action != proxywasm.ActionContinue {
		log.DefaultLogger.Errorf("[proxywasm][filter] OnReceive call ProxyOnRequestHeaders err: %v", err)
		return api.StreamFilterStop
	}

	endOfStream = 1
	if trailers != nil {
		endOfStream = 0
	}

	if buf != nil && buf.Len() > 0 {
		action, err = f.exports.ProxyOnRequestBody(f.contextID, int32(buf.Len()), int32(endOfStream))
		if err != nil || action != proxywasm.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter] OnReceive call ProxyOnRequestBody err: %v", err)
			return api.StreamFilterStop
		}
	}

	if trailers != nil {
		action, err = f.exports.ProxyOnRequestTrailers(f.contextID, int32(headerMapSize(trailers)))
		if err != nil || action != proxywasm.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter] OnReceive call ProxyOnRequestTrailers err: %v", err)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}

func (f *Filter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	f.instance.Lock(f.abi)
	defer f.instance.Unlock()

	endOfStream := 1
	if (buf != nil && buf.Len() > 0) || trailers != nil {
		endOfStream = 0
	}

	action, err := f.exports.ProxyOnResponseHeaders(f.contextID, int32(headerMapSize(headers)), int32(endOfStream))
	if err != nil || action != proxywasm.ActionContinue {
		log.DefaultLogger.Errorf("[proxywasm][filter] Append call ProxyOnResponseHeaders err: %v", err)
		return api.StreamFilterStop
	}

	endOfStream = 1
	if trailers != nil {
		endOfStream = 0
	}

	if buf != nil && buf.Len() > 0 {
		action, err = f.exports.ProxyOnResponseBody(f.contextID, int32(buf.Len()), int32(endOfStream))
		if err != nil || action != proxywasm.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter] Append call ProxyOnResponseBody err: %v", err)
			return api.StreamFilterStop
		}
	}

	if trailers != nil {
		action, err = f.exports.ProxyOnResponseTrailers(f.contextID, int32(headerMapSize(trailers)))
		if err != nil || action != proxywasm.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter] Append call ProxyOnResponseTrailers err: %v", err)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}

func (f *Filter) GetRootContextID() int32 {
	return f.rootContextID
}

func (f *Filter) GetVmConfig() common.IoBuffer {
	return f.factory.GetVmConfig()
}

func (f *Filter) GetPluginConfig() common.IoBuffer {
	return f.factory.GetPluginConfig()
}

func (f *Filter) GetHttpRequestHeader() common.HeaderMap {
	if f.receiverFilterHandler == nil {
		return nil
	}

	return &proxywasm010.HeaderMapWrapper{HeaderMap: f.receiverFilterHandler.GetRequestHeaders()}
}

func (f *Filter) GetHttpRequestBody() common.IoBuffer {
	if f.receiverFilterHandler == nil {
		return nil
	}

	return &proxywasm010.IoBufferWrapper{IoBuffer: f.receiverFilterHandler.GetRequestData()}
}

func (f *Filter) GetHttpRequestTrailer() common.HeaderMap {
	if f.receiverFilterHandler == nil {
		return nil
	}

	return &proxywasm010.HeaderMapWrapper{HeaderMap: f.receiverFilterHandler.GetRequestTrailers()}
}

func (f *Filter) GetHttpResponseHeader() common.HeaderMap {
	if f.senderFilterHandler == nil {
		return nil
	}

	return &proxywasm010.HeaderMapWrapper{HeaderMap: f.senderFilterHandler.GetResponseHeaders()}
}

func (f *Filter) GetHttpResponseBody() common.IoBuffer {
	if f.senderFilterHandler == nil {
		return nil
	}

	return &proxywasm010.IoBufferWrapper{IoBuffer: f.senderFilterHandler.GetResponseData()}
}

func (f *Filter) GetHttpResponseTrailer() common.HeaderMap {
	if f.senderFilterHandler == nil {
		return nil
	}

	return &proxywasm010.HeaderMapWrapper{HeaderMap: f.senderFilterHandler.GetResponseTrailers()}
}
