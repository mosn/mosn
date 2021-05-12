// Copyright 2020 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !proxytest

package rawhostcall

import (
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

//export proxy_log
func ProxyLog(logLevel types.LogLevel, messageData *byte, messageSize int) types.Status

//export proxy_send_local_response
func ProxySendLocalResponse(statusCode uint32, statusCodeDetailData *byte, statusCodeDetailsSize int,
	bodyData *byte, bodySize int, headersData *byte, headersSize int, grpcStatus int32) types.Status

//export proxy_get_shared_data
func ProxyGetSharedData(keyData *byte, keySize int, returnValueData **byte, returnValueSize *int, returnCas *uint32) types.Status

//export proxy_set_shared_data
func ProxySetSharedData(keyData *byte, keySize int, valueData *byte, valueSize int, cas uint32) types.Status

//export proxy_register_shared_queue
func ProxyRegisterSharedQueue(nameData *byte, nameSize int, returnID *uint32) types.Status

//export proxy_resolve_shared_queue
func ProxyResolveSharedQueue(vmIDData *byte, vmIDSize int, nameData *byte, nameSize int, returnID *uint32) types.Status

//export proxy_dequeue_shared_queue
func ProxyDequeueSharedQueue(queueID uint32, returnValueData **byte, returnValueSize *int) types.Status

//export proxy_enqueue_shared_queue
func ProxyEnqueueSharedQueue(queueID uint32, valueData *byte, valueSize int) types.Status

//export proxy_get_header_map_value
func ProxyGetHeaderMapValue(mapType types.MapType, keyData *byte, keySize int, returnValueData **byte, returnValueSize *int) types.Status

//export proxy_add_header_map_value
func ProxyAddHeaderMapValue(mapType types.MapType, keyData *byte, keySize int, valueData *byte, valueSize int) types.Status

//export proxy_replace_header_map_value
func ProxyReplaceHeaderMapValue(mapType types.MapType, keyData *byte, keySize int, valueData *byte, valueSize int) types.Status

//export proxy_remove_header_map_value
func ProxyRemoveHeaderMapValue(mapType types.MapType, keyData *byte, keySize int) types.Status

//export proxy_get_header_map_pairs
func ProxyGetHeaderMapPairs(mapType types.MapType, returnValueData **byte, returnValueSize *int) types.Status

//export proxy_set_header_map_pairs
func ProxySetHeaderMapPairs(mapType types.MapType, mapData *byte, mapSize int) types.Status

//export proxy_get_buffer_bytes
func ProxyGetBufferBytes(bt types.BufferType, start int, maxSize int, returnBufferData **byte, returnBufferSize *int) types.Status

//export proxy_set_buffer_bytes
func ProxySetBufferBytes(bt types.BufferType, start int, maxSize int, bufferData *byte, bufferSize int) types.Status

//export proxy_continue_stream
func ProxyContinueStream(streamType types.StreamType) types.Status

//export proxy_close_stream
func ProxyCloseStream(streamType types.StreamType) types.Status

//export proxy_http_call
func ProxyHttpCall(upstreamData *byte, upstreamSize int, headerData *byte, headerSize int,
	bodyData *byte, bodySize int, trailersData *byte, trailersSize int, timeout uint32, calloutIDPtr *uint32,
) types.Status

//export proxy_set_tick_period_milliseconds
func ProxySetTickPeriodMilliseconds(period uint32) types.Status

//export proxy_set_effective_context
func ProxySetEffectiveContext(contextID uint32) types.Status

//export proxy_done
func ProxyDone() types.Status

//export proxy_define_metric
func ProxyDefineMetric(metricType types.MetricType, metricNameData *byte, metricNameSize int, returnMetricIDPtr *uint32) types.Status

//export proxy_increment_metric
func ProxyIncrementMetric(metricID uint32, offset int64) types.Status

//export proxy_record_metric
func ProxyRecordMetric(metricID uint32, value uint64) types.Status

//export proxy_get_metric
func ProxyGetMetric(metricID uint32, returnMetricValue *uint64) types.Status

//export proxy_get_property
func ProxyGetProperty(pathData *byte, pathSize int, returnValueData **byte, returnValueSize *int) types.Status

//export proxy_set_property
func ProxySetProperty(pathData *byte, pathSize int, valueData *byte, valueSize int) types.Status
