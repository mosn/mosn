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

type DefaultImportsHandler struct{}

func (d *DefaultImportsHandler) Wait() Action { return ActionContinue }

func (d *DefaultImportsHandler) Log(logLevel LogLevel, msg string) Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) SetEffectiveContext(contextID int32) Result {
	return ResultUnimplemented
}

func (d *DefaultImportsHandler) ContextFinalize() Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) ResumeDownStream() Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) ResumeUpStream() Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) ResumeHttpRequest() Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) ResumeHttpResponse() Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) ResumeCustomStream(streamType StreamType) Result {
	return ResultUnimplemented
}

func (d *DefaultImportsHandler) CloseDownStream() Result { return ResultUnimplemented }
func (d *DefaultImportsHandler) CloseUpStream() Result   { return ResultUnimplemented }

func (d *DefaultImportsHandler) CloseHttpRequest() Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) CloseHttpResponse() Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) CloseCustomStream(streamType StreamType) Result {
	return ResultUnimplemented
}

func (d *DefaultImportsHandler) SendHttpResp(responseCode int32, responseCodeDetails common.IoBuffer,
	responseBody common.IoBuffer, additionalHeadersMap common.HeaderMap, grpcStatus int32) Result {
	return ResultUnimplemented
}

func (d *DefaultImportsHandler) GetHttpRequestBody() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetHttpResponseBody() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetDownStreamData() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetUpstreamData() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetHttpCalloutResponseBody() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetPluginConfig() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetVmConfig() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetCustomBuffer(bufferType BufferType) common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetHttpRequestHeader() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpRequestTrailer() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpRequestMetadata() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpResponseHeader() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpResponseTrailer() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpResponseMetadata() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpCallResponseHeaders() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpCallResponseTrailer() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpCallResponseMetadata() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetCustomMap(mapType MapType) common.HeaderMap { return nil }

func (d *DefaultImportsHandler) OpenSharedKvstore(kvstoreName string, createIfNotExist bool) (uint32, Result) {
	return 0, ResultUnimplemented
}
func (d *DefaultImportsHandler) GetSharedKvstore(kvstoreID uint32) KVStore { return nil }

func (d *DefaultImportsHandler) DeleteSharedKvstore(kvstoreID uint32) Result {
	return ResultUnimplemented
}

func (d *DefaultImportsHandler) OpenSharedQueue(queueName string, createIfNotExist bool) (uint32, Result) {
	return 0, ResultUnimplemented
}

func (d *DefaultImportsHandler) DequeueSharedQueueItem(queueID uint32) (string, Result) {
	return "", ResultUnimplemented
}

func (d *DefaultImportsHandler) EnqueueSharedQueueItem(queueID uint32, payload string) Result {
	return ResultUnimplemented
}

func (d *DefaultImportsHandler) DeleteSharedQueue(queueID uint32) Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) CreateTimer(period int32, oneTime bool) (uint32, Result) {
	return 0, ResultUnimplemented
}

func (d *DefaultImportsHandler) DeleteTimer(timerID uint32) Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) CreateMetric(metricType MetricType, metricName string) (uint32, Result) {
	return 0, ResultUnimplemented
}

func (d *DefaultImportsHandler) GetMetricValue(metricID uint32) (int64, Result) {
	return 0, ResultUnimplemented
}

func (d *DefaultImportsHandler) SetMetricValue(metricID uint32, value int64) Result {
	return ResultUnimplemented
}

func (d *DefaultImportsHandler) IncrementMetricValue(metricID uint32, offset int64) Result {
	return ResultUnimplemented
}

func (d *DefaultImportsHandler) DeleteMetric(metricID uint32) Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) DispatchHttpCall(upstream string, headersMap common.HeaderMap, bodyData common.IoBuffer,
	trailersMap common.HeaderMap, timeoutMilliseconds uint32) (uint32, Result) {
	return 0, ResultUnimplemented
}

func (d *DefaultImportsHandler) DispatchGrpcCall(upstream string, serviceName string, serviceMethod string,
	initialMetadataMap common.HeaderMap, grpcMessage common.IoBuffer, timeoutMilliseconds uint32) (uint32, Result) {
	return 0, ResultUnimplemented
}

func (d *DefaultImportsHandler) OpenGrpcStream(upstream string, serviceName string, serviceMethod string,
	initialMetadataMap common.HeaderMap) (uint32, Result) {
	return 0, ResultUnimplemented
}

func (d *DefaultImportsHandler) SendGrpcStreamMessage(calloutID uint32, grpcMessageData common.IoBuffer) Result {
	return ResultUnimplemented
}

func (d *DefaultImportsHandler) CancelGrpcCall(calloutID uint32) Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) CloseGrpcCall(calloutID uint32) Result { return ResultUnimplemented }

func (d *DefaultImportsHandler) CallCustomFunction(customFunctionID uint32, parametersData string) (string, Result) {
	return "", ResultUnimplemented
}
