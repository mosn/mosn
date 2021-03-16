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

type Action int32

const (
	ActionContinue Action = 0
	ActionPause    Action = 1
)

type MapType int32

const (
	MapTypeHttpRequestHeaders          MapType = 0
	MapTypeHttpRequestTrailers         MapType = 1
	MapTypeHttpResponseHeaders         MapType = 2
	MapTypeHttpResponseTrailers        MapType = 3
	MapTypeGrpcReceiveInitialMetadata  MapType = 4
	MapTypeGrpcReceiveTrailingMetadata MapType = 5
	MapTypeHttpCallResponseHeaders     MapType = 6
	MapTypeHttpCallResponseTrailers    MapType = 7
)

type BufferType int32

const (
	BufferTypeHttpRequestBody      BufferType = 0
	BufferTypeHttpResponseBody     BufferType = 1
	BufferTypeDownstreamData       BufferType = 2
	BufferTypeUpstreamData         BufferType = 3
	BufferTypeHttpCallResponseBody BufferType = 4
	BufferTypeGrpcReceiveBuffer    BufferType = 5
	BufferTypeVmConfiguration      BufferType = 6
	BufferTypePluginConfiguration  BufferType = 7
	BufferTypeCallData             BufferType = 8
)

type MetricType int32

const (
	MetricTypeCounter   MetricType = 0
	MetricTypeGauge     MetricType = 1
	MetricTypeHistogram MetricType = 2
	MetricTypeMax       MetricType = 2
)

type WasmResult int32

const (
	WasmResultOk                   WasmResult = 0
	WasmResultNotFound             WasmResult = 1  // The result could not be found, e.g. a provided key did not appear in a table.
	WasmResultBadArgument          WasmResult = 2  // An argument was bad, e.g. did not not conform to the required range.
	WasmResultSerializationFailure WasmResult = 3  // A protobuf could not be serialized.
	WasmResultParseFailure         WasmResult = 4  // A protobuf could not be parsed.
	WasmResultBadExpression        WasmResult = 5  // A provided expression (e.g. "foo.bar") was illegal or unrecognized.
	WasmResultInvalidMemoryAccess  WasmResult = 6  // A provided memory range was not legal.
	WasmResultEmpty                WasmResult = 7  // Data was requested from an empty container.
	WasmResultCasMismatch          WasmResult = 8  // The provided CAS did not match that of the stored data.
	WasmResultResultMismatch       WasmResult = 9  // Returned result was unexpected, e.g. of the incorrect size.
	WasmResultInternalFailure      WasmResult = 10 // Internal failure: trying check logs of the surrounding system.
	WasmResultBrokenConnection     WasmResult = 11 // The connection/stream/pipe was broken/closed unexpectedly.
	WasmResultUnimplemented        WasmResult = 12 // Feature not implemented.
)

func (wasmResult WasmResult) Int32() int32 {
	return int32(wasmResult)
}

type LogLevel uint8

const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelCritical
)
