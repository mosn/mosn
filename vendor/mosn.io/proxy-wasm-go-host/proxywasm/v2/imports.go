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

func RegisterImports(instance common.WasmInstance) {
	_ = instance.RegisterFunc("env", "proxy_log", ProxyLog)

	_ = instance.RegisterFunc("env", "proxy_set_effective_context", ProxySetEffectiveContext)
	_ = instance.RegisterFunc("env", "proxy_context_finalize", ProxyContextFinalize)

	_ = instance.RegisterFunc("env", "proxy_resume_stream", ProxyResumeStream)
	_ = instance.RegisterFunc("env", "proxy_close_stream", ProxyCloseStream)

	_ = instance.RegisterFunc("env", "proxy_send_http_response", ProxySendHttpResponse)
	_ = instance.RegisterFunc("env", "proxy_resume_http_stream", ProxyResumeHttpStream)
	_ = instance.RegisterFunc("env", "proxy_close_http_stream", ProxyCloseHttpStream)

	_ = instance.RegisterFunc("env", "proxy_get_buffer", ProxyGetBuffer)
	_ = instance.RegisterFunc("env", "proxy_set_buffer", ProxySetBuffer)

	_ = instance.RegisterFunc("env", "proxy_get_map_values", ProxyGetMapValues)
	_ = instance.RegisterFunc("env", "proxy_set_map_values", ProxySetMapValues)

	_ = instance.RegisterFunc("env", "proxy_open_shared_kvstore", ProxyOpenSharedKvstore)
	_ = instance.RegisterFunc("env", "proxy_get_shared_kvstore_key_values", ProxyGetSharedKvstoreKeyValues)
	_ = instance.RegisterFunc("env", "proxy_set_shared_kvstore_key_values", ProxySetSharedKvstoreKeyValues)
	_ = instance.RegisterFunc("env", "proxy_add_shared_kvstore_key_values", ProxyAddSharedKvstoreKeyValues)
	_ = instance.RegisterFunc("env", "proxy_remove_shared_kvstore_key", ProxyRemoveSharedKvstoreKey)
	_ = instance.RegisterFunc("env", "proxy_delete_shared_kvstore", ProxyDeleteSharedKvstore)

	_ = instance.RegisterFunc("env", "proxy_open_shared_queue", ProxyOpenSharedQueue)
	_ = instance.RegisterFunc("env", "proxy_dequeue_shared_queue_item", ProxyDequeueSharedQueueItem)
	_ = instance.RegisterFunc("env", "proxy_enqueue_shared_queue_item", ProxyEnqueueSharedQueueItem)
	_ = instance.RegisterFunc("env", "proxy_delete_shared_queue", ProxyDeleteSharedQueue)

	_ = instance.RegisterFunc("env", "proxy_create_timer", ProxyCreateTimer)
	_ = instance.RegisterFunc("env", "proxy_delete_timer", ProxyDeleteTimer)

	_ = instance.RegisterFunc("env", "proxy_create_metric", ProxyCreateMetric)
	_ = instance.RegisterFunc("env", "proxy_get_metric_value", ProxyGetMetricValue)
	_ = instance.RegisterFunc("env", "proxy_set_metric_value", ProxySetMetricValue)
	_ = instance.RegisterFunc("env", "proxy_increment_metric_value", ProxyIncrementMetricValue)
	_ = instance.RegisterFunc("env", "proxy_delete_metric", ProxyDeleteMetric)

	_ = instance.RegisterFunc("env", "proxy_dispatch_http_call", ProxyDispatchHttpCall)

	_ = instance.RegisterFunc("env", "proxy_dispatch_grpc_call", ProxyDispatchGrpcCall)
	_ = instance.RegisterFunc("env", "proxy_open_grpc_stream", ProxyOpenGrpcStream)
	_ = instance.RegisterFunc("env", "proxy_send_grpc_stream_message", ProxySendGrpcStreamMessage)
	_ = instance.RegisterFunc("env", "proxy_cancel_grpc_call", ProxyCancelGrpcCall)
	_ = instance.RegisterFunc("env", "proxy_close_grpc_call", ProxyCloseGrpcCall)

	_ = instance.RegisterFunc("env", "proxy_call_custom_function", ProxyCallCustomFunction)
}

func ProxyLog(instance common.WasmInstance, logLevel LogLevel, messageData int32, messageSize int32) Result {
	msg, err := instance.GetMemory(uint64(messageData), uint64(messageSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	callback := getImportHandler(instance)

	return callback.Log(logLevel, string(msg))
}

func ProxySetEffectiveContext(instance common.WasmInstance, contextID int32) Result {
	callback := getImportHandler(instance)
	return callback.SetEffectiveContext(contextID)
}

func ProxyContextFinalize(instance common.WasmInstance) Result {
	callback := getImportHandler(instance)
	return callback.ContextFinalize()
}

func ProxyResumeStream(instance common.WasmInstance, streamType StreamType) Result {
	callback := getImportHandler(instance)
	switch streamType {
	case StreamTypeDownstream:
		return callback.ResumeDownStream()
	case StreamTypeUpstream:
		return callback.ResumeUpStream()
	case StreamTypeHttpRequest:
		return callback.ResumeHttpRequest()
	case StreamTypeHttpResponse:
		return callback.ResumeHttpResponse()
	default:
		return callback.ResumeCustomStream(streamType)
	}
}

func ProxyCloseStream(instance common.WasmInstance, streamType StreamType) Result {
	callback := getImportHandler(instance)
	switch streamType {
	case StreamTypeDownstream:
		return callback.CloseDownStream()
	case StreamTypeUpstream:
		return callback.CloseUpStream()
	case StreamTypeHttpRequest:
		return callback.CloseHttpRequest()
	case StreamTypeHttpResponse:
		return callback.CloseHttpResponse()
	default:
		return callback.CloseCustomStream(streamType)
	}
}

func ProxySendHttpResponse(instance common.WasmInstance, responseCode int32, responseCodeDetailsData int32, responseCodeDetailsSize int32,
	responseBodyData int32, responseBodySize int32, additionalHeadersMapData int32, additionalHeadersSize int32,
	grpcStatus int32) Result {
	respCodeDetail, err := instance.GetMemory(uint64(responseCodeDetailsData), uint64(responseCodeDetailsSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	respBody, err := instance.GetMemory(uint64(responseBodyData), uint64(responseBodySize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	additionalHeaderMapData, err := instance.GetMemory(uint64(additionalHeadersMapData), uint64(additionalHeadersSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	additionalHeaderMap := common.DecodeMap(additionalHeaderMapData)

	callback := getImportHandler(instance)

	return callback.SendHttpResp(responseCode,
		common.NewIoBufferBytes(respCodeDetail),
		common.NewIoBufferBytes(respBody),
		common.CommonHeader(additionalHeaderMap), grpcStatus)
}

func ProxyResumeHttpStream(instance common.WasmInstance, streamType StreamType) Result {
	callback := getImportHandler(instance)
	switch streamType {
	case StreamTypeHttpRequest:
		return callback.ResumeHttpRequest()
	case StreamTypeHttpResponse:
		return callback.ResumeHttpResponse()
	}
	return ResultBadArgument
}

func ProxyCloseHttpStream(instance common.WasmInstance, streamType StreamType) Result {
	callback := getImportHandler(instance)
	switch streamType {
	case StreamTypeHttpRequest:
		return callback.CloseHttpRequest()
	case StreamTypeHttpResponse:
		return callback.CloseHttpResponse()
	}
	return ResultBadArgument
}

func GetBuffer(instance common.WasmInstance, bufferType BufferType) common.IoBuffer {
	im := getImportHandler(instance)

	switch bufferType {
	case BufferTypeHttpRequestBody:
		return im.GetHttpRequestBody()
	case BufferTypeHttpResponseBody:
		return im.GetHttpResponseBody()
	case BufferTypeDownstreamData:
		return im.GetDownStreamData()
	case BufferTypeUpstreamData:
		return im.GetUpstreamData()
	case BufferTypeHttpCalloutResponseBody:
		return im.GetHttpCalloutResponseBody()
	case BufferTypePluginConfiguration:
		return im.GetPluginConfig()
	case BufferTypeVmConfiguration:
		return im.GetVmConfig()
	default:
		return im.GetCustomBuffer(bufferType)
	}
}

func ProxyGetBuffer(instance common.WasmInstance, bufferType BufferType, offset int32, maxSize int32,
	returnBufferData int32, returnBufferSize int32) Result {
	buf := GetBuffer(instance, bufferType)
	if buf == nil {
		return ResultBadArgument
	}

	if buf.Len() == 0 {
		return ResultEmpty
	}

	if offset > offset+maxSize {
		return ResultBadArgument
	}

	if offset+maxSize > int32(buf.Len()) {
		maxSize = int32(buf.Len()) - offset
	}

	return copyIntoInstance(instance, buf.Bytes()[offset:offset+maxSize], returnBufferData, returnBufferSize)
}

func ProxySetBuffer(instance common.WasmInstance, bufferType BufferType, offset int32, size int32,
	bufferData int32, bufferSize int32) Result {
	buf := GetBuffer(instance, bufferType)
	if buf == nil {
		return ResultBadArgument
	}

	content, err := instance.GetMemory(uint64(bufferData), uint64(bufferSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	if offset == 0 {
		if size == 0 || int(size) >= buf.Len() {
			buf.Drain(buf.Len())
			_, err = buf.Write(content)
		} else {
			return ResultBadArgument
		}
	} else if int(offset) >= buf.Len() {
		_, err = buf.Write(content)
	} else {
		return ResultBadArgument
	}

	if err != nil {
		return ResultInvalidMemoryAccess
	}

	return ResultOk
}

func GetMap(instance common.WasmInstance, mapType MapType) common.HeaderMap {
	ctx := getImportHandler(instance)

	switch mapType {
	case MapTypeHttpRequestHeaders:
		return ctx.GetHttpRequestHeader()
	case MapTypeHttpRequestTrailers:
		return ctx.GetHttpRequestTrailer()
	case MapTypeHttpRequestMetadata:
		return ctx.GetHttpRequestMetadata()
	case MapTypeHttpResponseHeaders:
		return ctx.GetHttpResponseHeader()
	case MapTypeHttpResponseTrailers:
		return ctx.GetHttpResponseTrailer()
	case MapTypeHttpResponseMetadata:
		return ctx.GetHttpResponseMetadata()
	case MapTypeHttpCallResponseHeaders:
		return ctx.GetHttpCallResponseHeaders()
	case MapTypeHttpCallResponseTrailers:
		return ctx.GetHttpCallResponseTrailer()
	case MapTypeHttpCallResponseMetadata:
		return ctx.GetHttpCallResponseMetadata()
	default:
		return ctx.GetCustomMap(mapType)
	}
}

func copyMapIntoInstance(m common.HeaderMap, instance common.WasmInstance, returnMapData int32, returnMapSize int32) Result {
	cloneMap := make(map[string]string)
	totalBytesLen := 4
	m.Range(func(key, value string) bool {
		cloneMap[key] = value
		totalBytesLen += 4 + 4                         // keyLen + valueLen
		totalBytesLen += len(key) + 1 + len(value) + 1 // key + \0 + value + \0
		return true
	})

	addr, err := instance.Malloc(int32(totalBytesLen))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	err = instance.PutUint32(addr, uint32(len(cloneMap)))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	lenPtr := addr + 4
	dataPtr := lenPtr + uint64(8*len(cloneMap))

	for k, v := range cloneMap {
		_ = instance.PutUint32(lenPtr, uint32(len(k)))
		lenPtr += 4
		_ = instance.PutUint32(lenPtr, uint32(len(v)))
		lenPtr += 4

		_ = instance.PutMemory(dataPtr, uint64(len(k)), []byte(k))
		dataPtr += uint64(len(k))
		_ = instance.PutByte(dataPtr, 0)
		dataPtr++

		_ = instance.PutMemory(dataPtr, uint64(len(v)), []byte(v))
		dataPtr += uint64(len(v))
		_ = instance.PutByte(dataPtr, 0)
		dataPtr++
	}

	err = instance.PutUint32(uint64(returnMapData), uint32(addr))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	err = instance.PutUint32(uint64(returnMapSize), uint32(totalBytesLen))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	return ResultOk
}

func ProxyGetMapValues(instance common.WasmInstance, mapType MapType, keysData int32, keysSize int32,
	returnMapData int32, returnMapSize int32) Result {
	m := GetMap(instance, mapType)
	if m == nil {
		return ResultNotFound
	}

	if keysSize == 0 {
		// If the list of keys is empty, then all key-values in the map are returned.
		return copyMapIntoInstance(m, instance, returnMapData, returnMapSize)
	}

	key, err := instance.GetMemory(uint64(keysData), uint64(keysSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	if len(key) == 0 {
		return ResultBadArgument
	}

	value, exists := m.Get(string(key))
	if !exists {
		return ResultNotFound
	}

	return copyIntoInstance(instance, []byte(value), returnMapData, returnMapSize)
}

func ProxySetMapValues(instance common.WasmInstance, mapType MapType, removeKeysData int32, removeKeysSize int32,
	mapData int32, mapSize int32) Result {
	m := GetMap(instance, mapType)
	if m == nil {
		return ResultNotFound
	}

	// add new map data
	newMapContent, err := instance.GetMemory(uint64(mapData), uint64(mapSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	newMap := common.DecodeMap(newMapContent)

	for k, v := range newMap {
		m.Set(k, v)
	}

	// remove unwanted data
	key, err := instance.GetMemory(uint64(removeKeysData), uint64(removeKeysSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	if len(key) != 0 {
		m.Del(string(key))
	}

	return ResultOk
}

func ProxyOpenSharedKvstore(instance common.WasmInstance, kvstoreNameData int32, kvstoreNameSize int32, createIfNotExist int32,
	returnKvstoreID int32) Result {
	kvstoreName, err := instance.GetMemory(uint64(kvstoreNameData), uint64(kvstoreNameSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	if len(kvstoreName) == 0 {
		return ResultBadArgument
	}

	callback := getImportHandler(instance)

	kvStoreID, res := callback.OpenSharedKvstore(string(kvstoreName), intToBool(createIfNotExist))
	if res != ResultOk {
		return res
	}

	err = instance.PutUint32(uint64(returnKvstoreID), kvStoreID)
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	return ResultOk
}

func ProxyGetSharedKvstoreKeyValues(instance common.WasmInstance, kvstoreID int32, keyData int32, keySize int32,
	returnValuesData int32, returnValuesSize int32, returnCas int32) Result {
	callback := getImportHandler(instance)

	kvstore := callback.GetSharedKvstore(uint32(kvstoreID))
	if kvstore == nil {
		return ResultBadArgument
	}

	key, err := instance.GetMemory(uint64(keyData), uint64(keySize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	if len(key) == 0 {
		return ResultBadArgument
	}

	value, exists := kvstore.Get(string(key))
	if !exists {
		return ResultNotFound
	}

	return copyIntoInstance(instance, []byte(value), returnValuesData, returnValuesSize)
}

func ProxySetSharedKvstoreKeyValues(instance common.WasmInstance, kvstoreID int32, keyData int32, keySize int32,
	valuesData int32, valuesSize int32, cas int32) Result {
	callback := getImportHandler(instance)

	kvstore := callback.GetSharedKvstore(uint32(kvstoreID))
	if kvstore == nil {
		return ResultBadArgument
	}

	key, err := instance.GetMemory(uint64(keyData), uint64(keySize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	if len(key) == 0 {
		return ResultBadArgument
	}

	value, err := instance.GetMemory(uint64(valuesData), uint64(valuesSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	res := kvstore.SetCAS(string(key), string(value), intToBool(cas))
	if !res {
		return ResultCompareAndSwapMismatch
	}

	return ResultOk
}

func ProxyAddSharedKvstoreKeyValues(instance common.WasmInstance, kvstoreID int32, keyData int32, keySize int32,
	valuesData int32, valuesSize int32, cas int32) Result {
	callback := getImportHandler(instance)

	kvstore := callback.GetSharedKvstore(uint32(kvstoreID))
	if kvstore == nil {
		return ResultBadArgument
	}

	key, err := instance.GetMemory(uint64(keyData), uint64(keySize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	if len(key) == 0 {
		return ResultBadArgument
	}

	value, err := instance.GetMemory(uint64(valuesData), uint64(valuesSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	res := kvstore.SetCAS(string(key), string(value), intToBool(cas))
	if !res {
		return ResultCompareAndSwapMismatch
	}

	return ResultOk
}

func ProxyRemoveSharedKvstoreKey(instance common.WasmInstance, kvstoreID int32, keyData int32, keySize int32, cas int32) Result {
	callback := getImportHandler(instance)

	kvstore := callback.GetSharedKvstore(uint32(kvstoreID))
	if kvstore == nil {
		return ResultBadArgument
	}

	key, err := instance.GetMemory(uint64(keyData), uint64(keySize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	if len(key) == 0 {
		return ResultBadArgument
	}

	res := kvstore.DelCAS(string(key), intToBool(cas))
	if !res {
		return ResultCompareAndSwapMismatch
	}

	return ResultOk
}

func ProxyDeleteSharedKvstore(instance common.WasmInstance, kvstoreID int32) Result {
	callback := getImportHandler(instance)
	return callback.DeleteSharedKvstore(uint32(kvstoreID))
}

func ProxyOpenSharedQueue(instance common.WasmInstance, queueNameData int32, queueNameSize int32, createIfNotExist int32,
	returnQueueID int32) Result {
	queueName, err := instance.GetMemory(uint64(queueNameData), uint64(queueNameSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	if len(queueName) == 0 {
		return ResultBadArgument
	}

	callback := getImportHandler(instance)

	queueID, res := callback.OpenSharedQueue(string(queueName), intToBool(createIfNotExist))
	if res != ResultOk {
		return res
	}

	err = instance.PutUint32(uint64(returnQueueID), queueID)
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	return ResultOk
}

func ProxyDequeueSharedQueueItem(instance common.WasmInstance, queueID int32, returnPayloadData int32, returnPayloadSize int32) Result {
	callback := getImportHandler(instance)

	value, res := callback.DequeueSharedQueueItem(uint32(queueID))
	if res != ResultOk {
		return res
	}

	return copyIntoInstance(instance, []byte(value), returnPayloadData, returnPayloadSize)
}

func ProxyEnqueueSharedQueueItem(instance common.WasmInstance, queueID int32, payloadData int32, payloadSize int32) Result {
	value, err := instance.GetMemory(uint64(payloadData), uint64(payloadSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	callback := getImportHandler(instance)

	return callback.EnqueueSharedQueueItem(uint32(queueID), string(value))
}

func ProxyDeleteSharedQueue(instance common.WasmInstance, queueID int32) Result {
	callback := getImportHandler(instance)
	return callback.DeleteSharedQueue(uint32(queueID))
}

func ProxyCreateTimer(instance common.WasmInstance, period int32, oneTime int32, returnTimerID int32) Result {
	callback := getImportHandler(instance)

	timerID, res := callback.CreateTimer(period, intToBool(oneTime))
	if res != ResultOk {
		return res
	}

	err := instance.PutUint32(uint64(returnTimerID), timerID)
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	return ResultOk
}

func ProxyDeleteTimer(instance common.WasmInstance, timerID int32) Result {
	callback := getImportHandler(instance)
	return callback.DeleteTimer(uint32(timerID))
}

func ProxyCreateMetric(instance common.WasmInstance, metricType MetricType,
	metricNameData int32, metricNameSize int32, returnMetricID int32) Result {
	ctx := getImportHandler(instance)

	name, err := instance.GetMemory(uint64(metricNameData), uint64(metricNameSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	if len(name) == 0 {
		return ResultBadArgument
	}

	mid, res := ctx.CreateMetric(metricType, string(name))
	if res != ResultOk {
		return res
	}

	err = instance.PutUint32(uint64(returnMetricID), mid)
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	return ResultOk
}

func ProxyGetMetricValue(instance common.WasmInstance, metricID int32, returnValue int32) Result {
	ctx := getImportHandler(instance)

	value, res := ctx.GetMetricValue(uint32(metricID))
	if res != ResultOk {
		return res
	}

	err := instance.PutUint32(uint64(returnValue), uint32(value))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	return ResultOk
}

func ProxySetMetricValue(instance common.WasmInstance, metricID int32, value int64) Result {
	ctx := getImportHandler(instance)
	res := ctx.SetMetricValue(uint32(metricID), value)
	return res
}

func ProxyIncrementMetricValue(instance common.WasmInstance, metricID int32, offset int64) Result {
	ctx := getImportHandler(instance)
	return ctx.IncrementMetricValue(uint32(metricID), offset)
}

func ProxyDeleteMetric(instance common.WasmInstance, metricID int32) Result {
	ctx := getImportHandler(instance)
	return ctx.DeleteMetric(uint32(metricID))
}

func ProxyDispatchHttpCall(instance common.WasmInstance, upstreamNameData int32, upstreamNameSize int32, headersMapData int32, headersMapSize int32,
	bodyData int32, bodySize int32, trailersMapData int32, trailersMapSize int32, timeoutMilliseconds int32,
	returnCalloutID int32) Result {
	upstream, err := instance.GetMemory(uint64(upstreamNameData), uint64(upstreamNameSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	headerMapData, err := instance.GetMemory(uint64(headersMapData), uint64(headersMapSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	headerMap := common.DecodeMap(headerMapData)

	body, err := instance.GetMemory(uint64(bodyData), uint64(bodySize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	trailerMapData, err := instance.GetMemory(uint64(trailersMapData), uint64(trailersMapSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	trailerMap := common.DecodeMap(trailerMapData)

	ctx := getImportHandler(instance)

	calloutID, res := ctx.DispatchHttpCall(string(upstream),
		common.CommonHeader(headerMap), common.NewIoBufferBytes(body), common.CommonHeader(trailerMap),
		uint32(timeoutMilliseconds),
	)
	if res != ResultOk {
		return res
	}

	err = instance.PutUint32(uint64(returnCalloutID), calloutID)
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	return ResultOk
}

func ProxyDispatchGrpcCall(instance common.WasmInstance, upstreamNameData int32, upstreamNameSize int32, serviceNameData int32, serviceNameSize int32,
	serviceMethodData int32, serviceMethodSize int32, initialMetadataMapData int32, initialMetadataMapSize int32,
	grpcMessageData int32, grpcMessageSize int32, timeoutMilliseconds int32, returnCalloutID int32) Result {
	upstream, err := instance.GetMemory(uint64(upstreamNameData), uint64(upstreamNameSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	serviceName, err := instance.GetMemory(uint64(serviceNameData), uint64(serviceNameSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	serviceMethod, err := instance.GetMemory(uint64(serviceMethodData), uint64(serviceMethodSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	initialMetaMapdata, err := instance.GetMemory(uint64(initialMetadataMapData), uint64(initialMetadataMapSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	initialMetadataMap := common.DecodeMap(initialMetaMapdata)

	msg, err := instance.GetMemory(uint64(grpcMessageData), uint64(grpcMessageSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	ctx := getImportHandler(instance)

	calloutID, res := ctx.DispatchGrpcCall(string(upstream), string(serviceName), string(serviceMethod),
		common.CommonHeader(initialMetadataMap), common.NewIoBufferBytes(msg), uint32(timeoutMilliseconds))
	if res != ResultOk {
		return res
	}

	err = instance.PutUint32(uint64(returnCalloutID), calloutID)
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	return ResultOk
}

func ProxyOpenGrpcStream(instance common.WasmInstance, upstreamNameData int32, upstreamNameSize int32, serviceNameData int32, serviceNameSize int32,
	serviceMethodData int32, serviceMethodSize int32, initialMetadataMapData int32, initialMetadataMapSize int32,
	returnCalloutID int32) Result {
	upstream, err := instance.GetMemory(uint64(upstreamNameData), uint64(upstreamNameSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	serviceName, err := instance.GetMemory(uint64(serviceNameData), uint64(serviceNameSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	serviceMethod, err := instance.GetMemory(uint64(serviceMethodData), uint64(serviceMethodSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	initialMetaMapdata, err := instance.GetMemory(uint64(initialMetadataMapData), uint64(initialMetadataMapSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}
	initialMetadataMap := common.DecodeMap(initialMetaMapdata)

	ctx := getImportHandler(instance)

	calloutID, res := ctx.OpenGrpcStream(string(upstream), string(serviceName), string(serviceMethod), common.CommonHeader(initialMetadataMap))
	if res != ResultOk {
		return res
	}

	err = instance.PutUint32(uint64(returnCalloutID), calloutID)
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	return ResultOk
}

func ProxySendGrpcStreamMessage(instance common.WasmInstance, calloutID int32, grpcMessageData int32, grpcMessageSize int32) Result {
	grpcMessage, err := instance.GetMemory(uint64(grpcMessageData), uint64(grpcMessageSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	ctx := getImportHandler(instance)
	return ctx.SendGrpcStreamMessage(uint32(calloutID), common.NewIoBufferBytes(grpcMessage))
}

func ProxyCancelGrpcCall(instance common.WasmInstance, calloutID int32) Result {
	callback := getImportHandler(instance)
	return callback.CancelGrpcCall(uint32(calloutID))
}

func ProxyCloseGrpcCall(instance common.WasmInstance, calloutID int32) Result {
	callback := getImportHandler(instance)
	return callback.CloseGrpcCall(uint32(calloutID))
}

func ProxyCallCustomFunction(instance common.WasmInstance, customFunctionID int32, parametersData int32, parametersSize int32,
	returnResultsData int32, returnResultsSize int32) Result {

	param, err := instance.GetMemory(uint64(parametersData), uint64(parametersSize))
	if err != nil {
		return ResultInvalidMemoryAccess
	}

	ctx := getImportHandler(instance)

	ret, res := ctx.CallCustomFunction(uint32(customFunctionID), string(param))
	if res != ResultOk {
		return res
	}

	return copyIntoInstance(instance, []byte(ret), returnResultsData, returnResultsSize)
}
