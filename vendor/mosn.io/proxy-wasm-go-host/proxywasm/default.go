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
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/proxy-wasm-go-host/common"
)

type DefaultImportsHandler struct{}

// for golang host environment, no-op
func (d *DefaultImportsHandler) Wait() {}

// utils
func (d *DefaultImportsHandler) GetRootContextID() int32 { return 0 }

func (d *DefaultImportsHandler) GetVmConfig() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetPluginConfig() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) Log(level LogLevel, msg string) WasmResult {
	fmt.Println(msg)
	return WasmResultOk
}

func (d *DefaultImportsHandler) SetEffectiveContextID(contextID int32) WasmResult {
	return WasmResultUnimplemented
}

func (d *DefaultImportsHandler) SetTickPeriodMilliseconds(tickPeriodMilliseconds int32) WasmResult {
	return WasmResultUnimplemented
}

func (d *DefaultImportsHandler) GetCurrentTimeNanoseconds() (int32, WasmResult) {
	nano := time.Now().Nanosecond()
	return int32(nano), WasmResultOk
}

func (d *DefaultImportsHandler) Done() WasmResult { return WasmResultUnimplemented }

// l4

func (d *DefaultImportsHandler) GetDownStreamData() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetUpstreamData() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) ResumeDownstream() WasmResult { return WasmResultUnimplemented }

func (d *DefaultImportsHandler) ResumeUpstream() WasmResult { return WasmResultUnimplemented }

// http

func (d *DefaultImportsHandler) GetHttpRequestHeader() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpRequestBody() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetHttpRequestTrailer() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpResponseHeader() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpResponseBody() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetHttpResponseTrailer() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpCallResponseHeaders() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetHttpCallResponseBody() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetHttpCallResponseTrailer() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) HttpCall(url string, headers common.HeaderMap, body common.IoBuffer, trailer common.HeaderMap, timeoutMilliseconds int32) (int32, WasmResult) {
	return 0, WasmResultUnimplemented
}

func (d *DefaultImportsHandler) ResumeHttpRequest() WasmResult { return WasmResultUnimplemented }

func (d *DefaultImportsHandler) ResumeHttpResponse() WasmResult { return WasmResultUnimplemented }

func (d *DefaultImportsHandler) SendHttpResp(respCode int32, respCodeDetail common.IoBuffer, respBody common.IoBuffer, additionalHeaderMap common.HeaderMap, grpcCode int32) WasmResult {
	return WasmResultUnimplemented
}

// grpc

func (d *DefaultImportsHandler) OpenGrpcStream(grpcService string, serviceName string, method string) (int32, WasmResult) {
	return 0, WasmResultUnimplemented
}

func (d *DefaultImportsHandler) SendGrpcCallMsg(token int32, data common.IoBuffer, endOfStream int32) WasmResult {
	return WasmResultUnimplemented
}

func (d *DefaultImportsHandler) CancelGrpcCall(token int32) WasmResult {
	return WasmResultUnimplemented
}

func (d *DefaultImportsHandler) CloseGrpcCall(token int32) WasmResult { return WasmResultUnimplemented }

func (d *DefaultImportsHandler) GrpcCall(grpcService string, serviceName string, method string, data common.IoBuffer, timeoutMilliseconds int32) (int32, WasmResult) {
	return 0, WasmResultUnimplemented
}

func (d *DefaultImportsHandler) GetGrpcReceiveInitialMetaData() common.HeaderMap { return nil }

func (d *DefaultImportsHandler) GetGrpcReceiveBuffer() common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetGrpcReceiveTrailerMetaData() common.HeaderMap { return nil }

// foreign

func (d *DefaultImportsHandler) CallForeignFunction(funcName string, param string) (string, WasmResult) {
	return "", WasmResultUnimplemented
}

func (d *DefaultImportsHandler) GetFuncCallData() common.IoBuffer { return nil }

// property

func (d *DefaultImportsHandler) GetProperty(key string) (string, WasmResult) {
	return "", WasmResultOk
}

func (d *DefaultImportsHandler) SetProperty(key string, value string) WasmResult {
	return WasmResultUnimplemented
}

// metric

func (d *DefaultImportsHandler) DefineMetric(metricType MetricType, name string) (int32, WasmResult) {
	return 0, WasmResultUnimplemented
}

func (d *DefaultImportsHandler) IncrementMetric(metricID int32, offset int64) WasmResult {
	return WasmResultUnimplemented
}

func (d *DefaultImportsHandler) RecordMetric(metricID int32, value int64) WasmResult {
	return WasmResultUnimplemented
}

func (d *DefaultImportsHandler) GetMetric(metricID int32) (int64, WasmResult) {
	return 0, WasmResultUnimplemented
}

func (d *DefaultImportsHandler) RemoveMetric(metricID int32) WasmResult {
	return WasmResultUnimplemented
}

// shared data

type sharedDataItem struct {
	data string
	cas  uint32
}

type sharedData struct {
	lock sync.RWMutex
	m    map[string]*sharedDataItem
	cas  uint32
}

func (s *sharedData) get(key string) (string, uint32, WasmResult) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if v, ok := s.m[key]; ok {
		return v.data, v.cas, WasmResultOk
	}

	return "", 0, WasmResultNotFound
}

func (s *sharedData) set(key string, value string, cas uint32) WasmResult {
	if key == "" {
		return WasmResultBadArgument
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if v, ok := s.m[key]; ok {
		if v.cas != cas {
			return WasmResultCasMismatch
		}
		v.data = value
		v.cas = atomic.AddUint32(&s.cas, 1)
		return WasmResultOk
	}

	s.m[key] = &sharedDataItem{
		data: value,
		cas:  atomic.AddUint32(&s.cas, 1),
	}

	return WasmResultOk
}

var globalSharedData = &sharedData{
	m: make(map[string]*sharedDataItem),
}

func (d *DefaultImportsHandler) GetSharedData(key string) (string, uint32, WasmResult) {
	return globalSharedData.get(key)
}

func (d *DefaultImportsHandler) SetSharedData(key string, value string, cas uint32) WasmResult {
	return globalSharedData.set(key, value, cas)
}

// shared queue

type sharedQueue struct {
	id    uint32
	name  string
	lock  sync.RWMutex
	queue []string
}

func (s *sharedQueue) enque(value string) WasmResult {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.queue = append(s.queue, value)

	return WasmResultOk
}

func (s *sharedQueue) deque() (string, WasmResult) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if len(s.queue) == 0 {
		return "", WasmResultEmpty
	}

	v := s.queue[0]
	s.queue = s.queue[1:]

	return v, WasmResultOk
}

type sharedQueueRegistry struct {
	lock             sync.RWMutex
	nameToIDMap      map[string]uint32
	m                map[uint32]*sharedQueue
	queueIDGenerator uint32
}

func (s *sharedQueueRegistry) register(queueName string) (uint32, WasmResult) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if queueID, ok := s.nameToIDMap[queueName]; ok {
		return queueID, WasmResultOk
	}

	newQueueID := atomic.AddUint32(&s.queueIDGenerator, 1)
	s.nameToIDMap[queueName] = newQueueID
	s.m[newQueueID] = &sharedQueue{
		id:    newQueueID,
		name:  queueName,
		queue: make([]string, 0),
	}

	return newQueueID, WasmResultOk
}

func (s *sharedQueueRegistry) delete(queueID uint32) WasmResult {
	s.lock.Lock()
	defer s.lock.Unlock()

	queue, ok := s.m[queueID]
	if !ok {
		return WasmResultNotFound
	}

	delete(s.nameToIDMap, queue.name)
	delete(s.m, queueID)

	return WasmResultOk
}

func (s *sharedQueueRegistry) resolve(queueName string) (uint32, WasmResult) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if queueID, ok := s.nameToIDMap[queueName]; ok {
		return queueID, WasmResultOk
	}

	return 0, WasmResultNotFound
}

func (s *sharedQueueRegistry) get(queueID uint32) *sharedQueue {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if queue, ok := s.m[queueID]; ok {
		return queue
	}

	return nil
}

var globalSharedQueueRegistry = &sharedQueueRegistry{
	nameToIDMap: make(map[string]uint32),
	m:           make(map[uint32]*sharedQueue),
}

func (d *DefaultImportsHandler) RegisterSharedQueue(queueName string) (uint32, WasmResult) {
	return globalSharedQueueRegistry.register(queueName)
}

func (d *DefaultImportsHandler) RemoveSharedQueue(queueID uint32) WasmResult {
	return globalSharedQueueRegistry.delete(queueID)
}

func (d *DefaultImportsHandler) ResolveSharedQueue(queueName string) (uint32, WasmResult) {
	return globalSharedQueueRegistry.resolve(queueName)
}

func (d *DefaultImportsHandler) EnqueueSharedQueue(queueID uint32, data string) WasmResult {
	queue := globalSharedQueueRegistry.get(queueID)
	if queue == nil {
		return WasmResultNotFound
	}

	return queue.enque(data)
}

func (d *DefaultImportsHandler) DequeueSharedQueue(queueID uint32) (string, WasmResult) {
	queue := globalSharedQueueRegistry.get(queueID)
	if queue == nil {
		return "", WasmResultNotFound
	}

	return queue.deque()
}

// custom extension

func (d *DefaultImportsHandler) GetCustomBuffer(bufferType BufferType) common.IoBuffer { return nil }

func (d *DefaultImportsHandler) GetCustomHeader(mapType MapType) common.HeaderMap { return nil }
