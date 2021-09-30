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

type ImportsHandler interface {
	// for golang host environment
	// Wait until async call return, eg. sync http call in golang
	Wait() Action

	// integration
	Log(logLevel LogLevel, msg string) Result

	SetEffectiveContext(contextID int32) Result
	ContextFinalize() Result

	// configuration
	GetPluginConfig() common.IoBuffer
	GetVmConfig() common.IoBuffer

	// buffer & map
	GetCustomBuffer(bufferType BufferType) common.IoBuffer
	GetCustomMap(mapType MapType) common.HeaderMap

	// l4
	GetDownStreamData() common.IoBuffer
	GetUpstreamData() common.IoBuffer

	ResumeDownStream() Result
	ResumeUpStream() Result

	CloseDownStream() Result
	CloseUpStream() Result

	ResumeCustomStream(streamType StreamType) Result
	CloseCustomStream(streamType StreamType) Result

	// http
	GetHttpRequestHeader() common.HeaderMap
	GetHttpRequestBody() common.IoBuffer
	GetHttpRequestTrailer() common.HeaderMap
	GetHttpRequestMetadata() common.HeaderMap

	GetHttpResponseHeader() common.HeaderMap
	GetHttpResponseBody() common.IoBuffer
	GetHttpResponseTrailer() common.HeaderMap
	GetHttpResponseMetadata() common.HeaderMap

	GetHttpCallResponseHeaders() common.HeaderMap
	GetHttpCalloutResponseBody() common.IoBuffer
	GetHttpCallResponseTrailer() common.HeaderMap
	GetHttpCallResponseMetadata() common.HeaderMap

	SendHttpResp(responseCode int32, responseCodeDetails common.IoBuffer, responseBody common.IoBuffer,
		additionalHeadersMap common.HeaderMap, grpcStatus int32) Result
	DispatchHttpCall(upstream string, headersMap common.HeaderMap, bodyData common.IoBuffer,
		trailersMap common.HeaderMap, timeoutMilliseconds uint32) (uint32, Result)

	ResumeHttpRequest() Result
	ResumeHttpResponse() Result

	CloseHttpRequest() Result
	CloseHttpResponse() Result

	// kv store
	OpenSharedKvstore(kvstoreName string, createIfNotExist bool) (uint32, Result)
	GetSharedKvstore(kvstoreID uint32) KVStore
	DeleteSharedKvstore(kvstoreID uint32) Result

	// shared queue
	OpenSharedQueue(queueName string, createIfNotExist bool) (uint32, Result)
	DequeueSharedQueueItem(queueID uint32) (string, Result)
	EnqueueSharedQueueItem(queueID uint32, payload string) Result
	DeleteSharedQueue(queueID uint32) Result

	// timer
	CreateTimer(period int32, oneTime bool) (uint32, Result)
	DeleteTimer(timerID uint32) Result

	// metric
	CreateMetric(metricType MetricType, metricName string) (uint32, Result)
	GetMetricValue(metricID uint32) (int64, Result)
	SetMetricValue(metricID uint32, value int64) Result
	IncrementMetricValue(metricID uint32, offset int64) Result
	DeleteMetric(metricID uint32) Result

	// grpc
	DispatchGrpcCall(upstream string, serviceName string, serviceMethod string, initialMetadataMap common.HeaderMap,
		grpcMessage common.IoBuffer, timeoutMilliseconds uint32) (uint32, Result)
	OpenGrpcStream(upstream string, serviceName string, serviceMethod string, initialMetadataMap common.HeaderMap) (uint32, Result)
	SendGrpcStreamMessage(calloutID uint32, grpcMessageData common.IoBuffer) Result
	CancelGrpcCall(calloutID uint32) Result
	CloseGrpcCall(calloutID uint32) Result

	// custom function
	CallCustomFunction(customFunctionID uint32, parametersData string) (string, Result)
}

type Exports interface {
	// integration
	ProxyOnMemoryAllocate(memorySize int32) (int32, error)

	// context
	ProxyOnContextCreate(contextID int32, parentContextID int32, contextType ContextType) (int32, error)
	ProxyOnContextFinalize(contextID int32) (int32, error)

	// configuration
	ProxyOnVmStart(vmID int32, vmConfigurationSize int32) (int32, error)
	ProxyOnPluginStart(pluginID int32, pluginConfigurationSize int32) (int32, error)

	// l4
	ProxyOnNewConnection(streamID int32) (Action, error)
	ProxyOnDownstreamData(streamID int32, dataSize int32, endOfStream int32) (Action, error)
	ProxyOnDownstreamClose(contextID int32, closeSource CloseSourceType) error
	ProxyOnUpstreamData(streamID int32, dataSize int32, endOfStream int32) (Action, error)
	ProxyOnUpstreamClose(streamID int32, closeSource CloseSourceType) error

	// http
	ProxyOnHttpRequestHeaders(streamID int32, numHeaders int32, endOfStream int32) (Action, error)
	ProxyOnHttpRequestBody(streamID int32, bodySize int32, endOfStream int32) (Action, error)
	ProxyOnHttpRequestTrailers(streamID int32, numTrailers int32, endOfStream int32) (Action, error)
	ProxyOnHttpRequestMetadata(streamID int32, numElements int32) (Action, error)

	ProxyOnHttpResponseHeaders(streamID int32, numHeaders int32, endOfStream int32) (Action, error)
	ProxyOnHttpResponseBody(streamID int32, bodySize int32, endOfStream int32) (Action, error)
	ProxyOnHttpResponseTrailers(streamID int32, numTrailers int32, endOfStream int32) (Action, error)
	ProxyOnHttpResponseMetadata(streamID int32, numElements int32) (Action, error)

	ProxyOnHttpCallResponse(calloutID int32, numHeaders int32, bodySize int32, numTrailers int32) error

	// queue
	ProxyOnQueueReady(queueID int32) error

	// timer
	ProxyOnTimerReady(timerID int32) error

	// grpc
	ProxyOnGrpcCallResponseHeaderMetadata(calloutID int32, numHeaders int32) error
	ProxyOnGrpcCallResponseMessage(calloutID int32, messageSize int32) error
	ProxyOnGrpcCallResponseTrailerMetadata(calloutID int32, numTrailers int32) error
	ProxyOnGrpcCallClose(calloutID int32, statusCode int32) error

	// custom function
	ProxyOnCustomCallback(customCallbackID int32, parametersSize int32) (int32, error)
}
