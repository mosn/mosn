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

package proxywasm_0_1_0

import (
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm/abi"
)

const ProxyWasmABI_0_1_0 string = "proxy_abi_version_0_1_0"

func init() {
	abi.RegisterABI(ProxyWasmABI_0_1_0, abiContextFactory)
}

func abiContextFactory(instance types.WasmInstance) types.ABI {
	return &abiContext{
		instance: instance,
		imports:  &DefaultImportsHandler{},
	}
}

type abiContext struct {
	imports     ImportsHandler
	instance    types.WasmInstance
	httpCallout *httpStruct
}

func (a *abiContext) Name() string {
	return ProxyWasmABI_0_1_0
}

func (a *abiContext) GetExports() interface{} {
	return a
}

func (a *abiContext) SetImports(imports interface{}) {
	cb, ok := imports.(ImportsHandler)
	if !ok {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][context] SetImports type is not ImportsHandler")
		return
	}

	a.imports = cb
}

func (a *abiContext) SetInstance(instance types.WasmInstance) {
	a.instance = instance
}

func (a *abiContext) OnInstanceCreate(instance types.WasmInstance) {
	_ = instance.RegisterFunc("env", "proxy_log", proxyLog)

	_ = instance.RegisterFunc("env", "proxy_set_effective_context", proxySetEffectiveContext)

	_ = instance.RegisterFunc("env", "proxy_get_property", proxyGetProperty)
	_ = instance.RegisterFunc("env", "proxy_set_property", proxySetProperty)

	_ = instance.RegisterFunc("env", "proxy_get_buffer_bytes", proxyGetBufferBytes)
	_ = instance.RegisterFunc("env", "proxy_set_buffer_bytes", proxySetBufferBytes)

	_ = instance.RegisterFunc("env", "proxy_get_header_map_pairs", proxyGetHeaderMapPairs)
	_ = instance.RegisterFunc("env", "proxy_set_header_map_pairs", proxySetHeaderMapPairs)

	_ = instance.RegisterFunc("env", "proxy_get_header_map_value", proxyGetHeaderMapValue)
	_ = instance.RegisterFunc("env", "proxy_replace_header_map_value", proxyReplaceHeaderMapValue)
	_ = instance.RegisterFunc("env", "proxy_add_header_map_value", proxyAddHeaderMapValue)
	_ = instance.RegisterFunc("env", "proxy_remove_header_map_value", proxyRemoveHeaderMapValue)

	_ = instance.RegisterFunc("env", "proxy_set_tick_period_milliseconds", proxySetTickPeriodMilliseconds)
	_ = instance.RegisterFunc("env", "proxy_get_current_time_nanoseconds", proxyGetCurrentTimeNanoseconds)

	_ = instance.RegisterFunc("env", "proxy_grpc_call", proxyGrpcCall)
	_ = instance.RegisterFunc("env", "proxy_grpc_stream", proxyGrpcStream)
	_ = instance.RegisterFunc("env", "proxy_grpc_cancel", proxyGrpcCancel)
	_ = instance.RegisterFunc("env", "proxy_grpc_close", proxyGrpcClose)
	_ = instance.RegisterFunc("env", "proxy_grpc_send", proxyGrpcSend)

	_ = instance.RegisterFunc("env", "proxy_http_call", proxyHttpCall)

	_ = instance.RegisterFunc("env", "proxy_define_metric", proxyDefineMetric)
	_ = instance.RegisterFunc("env", "proxy_increment_metric", proxyIncrementMetric)
	_ = instance.RegisterFunc("env", "proxy_record_metric", proxyRecordMetric)
	_ = instance.RegisterFunc("env", "proxy_get_metric", proxyGetMetric)

	_ = instance.RegisterFunc("env", "proxy_register_shared_queue", proxyRegisterSharedQueue)
	_ = instance.RegisterFunc("env", "proxy_resolve_shared_queue", proxyResolveSharedQueue)
	_ = instance.RegisterFunc("env", "proxy_dequeue_shared_queue", proxyDequeueSharedQueue)
	_ = instance.RegisterFunc("env", "proxy_enqueue_shared_queue", proxyEnqueueSharedQueue)

	_ = instance.RegisterFunc("env", "proxy_get_shared_data", proxyGetSharedData)
	_ = instance.RegisterFunc("env", "proxy_set_shared_data", proxySetSharedData)
}

func (a *abiContext) OnInstanceStart(instance types.WasmInstance) {}

func (a *abiContext) OnInstanceDestroy(instance types.WasmInstance) {}
