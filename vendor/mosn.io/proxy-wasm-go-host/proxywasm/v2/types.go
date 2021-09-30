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

package v2

import "mosn.io/proxy-wasm-go-host/proxywasm/common"

type KVStore interface {
	common.HeaderMap
	SetCAS(key, value string, cas bool) bool
	DelCAS(key string, cas bool) bool
}

type Result int32

const (
	ResultOk                     Result = 0
	ResultEmpty                  Result = 1
	ResultNotFound               Result = 2
	ResultNotAllowed             Result = 3
	ResultBadArgument            Result = 4
	ResultInvalidMemoryAccess    Result = 5
	ResultInvalidOperation       Result = 6
	ResultCompareAndSwapMismatch Result = 7
	ResultUnimplemented          Result = 12
)

type Action int32

const (
	ActionContinue         Action = 1
	ActionEndStream        Action = 2
	ActionDone             Action = 3
	ActionPause            Action = 4
	ActionWaitForMoreData  Action = 5
	ActionWaitForEndOrFull Action = 6
	ActionClose            Action = 7
)

type MetricType int32

const (
	MetricTypeCounter   MetricType = 1
	MetricTypeGauge     MetricType = 2
	MetricTypeHistogram MetricType = 3
)

type MapType int32

const (
	MapTypeHttpRequestHeaders       MapType = 1
	MapTypeHttpRequestTrailers      MapType = 2
	MapTypeHttpRequestMetadata      MapType = 3
	MapTypeHttpResponseHeaders      MapType = 4
	MapTypeHttpResponseTrailers     MapType = 5
	MapTypeHttpResponseMetadata     MapType = 6
	MapTypeHttpCallResponseHeaders  MapType = 7
	MapTypeHttpCallResponseTrailers MapType = 8
	MapTypeHttpCallResponseMetadata MapType = 9
)

type BufferType int32

const (
	BufferTypeVmConfiguration         BufferType = 1
	BufferTypePluginConfiguration     BufferType = 2
	BufferTypeDownstreamData          BufferType = 3
	BufferTypeUpstreamData            BufferType = 4
	BufferTypeHttpRequestBody         BufferType = 5
	BufferTypeHttpResponseBody        BufferType = 6
	BufferTypeHttpCalloutResponseBody BufferType = 7
)

type StreamType int32

const (
	StreamTypeDownstream   StreamType = 1
	StreamTypeUpstream     StreamType = 2
	StreamTypeHttpRequest  StreamType = 3
	StreamTypeHttpResponse StreamType = 4
)

type ContextType int32

const (
	ContextTypeVmContext     ContextType = 1
	ContextTypePluginContext ContextType = 2
	ContextTypeStreamContext ContextType = 3
	ContextTypeHttpContext   ContextType = 4
)

type LogLevel int32

const (
	LogLevelTrace   LogLevel = 1
	LogLevelDebug   LogLevel = 2
	LogLevelInfo    LogLevel = 3
	LogLevelWarning LogLevel = 4
	LogLevelError   LogLevel = 5
)

type CloseSourceType int32

const (
	CloseSourceTypeLocal  CloseSourceType = 1
	CloseSourceTypeRemote CloseSourceType = 2
)
