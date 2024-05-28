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
	wasmABI "mosn.io/mosn/pkg/wasm/abi"
	"mosn.io/pkg/buffer"
	v1Host "mosn.io/proxy-wasm-go-host/proxywasm/v1"
	v2Host "mosn.io/proxy-wasm-go-host/proxywasm/v2"
)

type Filter struct {
	ctx context.Context

	factory *FilterConfigFactory

	pluginName string
	plugin     types.WasmPlugin
	instance   types.WasmInstance
	abi        types.ABI
	exports    *exportAdapter

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

	filter.abi = wasmABI.GetABIList(instance)[0]
	log.DefaultLogger.Infof("[proxywasm][filter] abi version: %v", filter.abi.Name())
	if filter.abi.Name() == v1Host.ProxyWasmABI_0_1_0 {
		// v1
		imports := &v1Imports{factory: filter.factory, filter: filter}
		imports.DefaultImportsHandler.Instance = instance
		filter.abi.SetABIImports(imports)
		filter.exports = &exportAdapter{v1: filter.abi.GetABIExports().(v1Host.Exports)}
	} else if filter.abi.Name() == v2Host.ProxyWasmABI_0_2_0 {
		// v2
		imports := &v2Imports{factory: filter.factory, filter: filter}
		imports.DefaultImportsHandler.Instance = instance
		filter.abi.SetABIImports(imports)
		filter.exports = &exportAdapter{v2: filter.abi.GetABIExports().(v2Host.Exports)}
	} else {
		log.DefaultLogger.Errorf("[proxywasm][filter] unknown abi list: %v", filter.abi)
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
			// warn instead of error as some proxy_abi_version_0_1_0 wasm don't
			// export proxy_on_delete
			log.DefaultLogger.Warnf("[proxywasm][filter] OnDestroy fail to call ProxyOnDelete, err: %v", err)
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

	status := f.exports.ProxyOnRequestHeaders(f.contextID, int32(headerMapSize(headers)), int32(endOfStream))
	if status == api.StreamFilterStop {
		return api.StreamFilterStop
	}

	endOfStream = 1
	if trailers != nil {
		endOfStream = 0
	}

	if buf != nil && buf.Len() > 0 {
		status = f.exports.ProxyOnRequestBody(f.contextID, int32(buf.Len()), int32(endOfStream))
		if status == api.StreamFilterStop {
			return api.StreamFilterStop
		}
	}

	if trailers != nil {
		status = f.exports.ProxyOnRequestTrailers(f.contextID, int32(headerMapSize(trailers)), int32(endOfStream))
		if status == api.StreamFilterStop {
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

	status := f.exports.ProxyOnResponseHeaders(f.contextID, int32(headerMapSize(headers)), int32(endOfStream))
	if status == api.StreamFilterStop {
		return api.StreamFilterStop
	}

	endOfStream = 1
	if trailers != nil {
		endOfStream = 0
	}

	if buf != nil && buf.Len() > 0 {
		status = f.exports.ProxyOnResponseBody(f.contextID, int32(buf.Len()), int32(endOfStream))
		if status == api.StreamFilterStop {
			return api.StreamFilterStop
		}
	}

	if trailers != nil {
		status = f.exports.ProxyOnResponseTrailers(f.contextID, int32(headerMapSize(trailers)), int32(endOfStream))
		if status == api.StreamFilterStop {
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}
