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
	"context"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/buffer"
)

func getInstanceCallback(instance types.WasmInstance) ImportsHandler {
	v := instance.GetData()
	if v == nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] getInstanceCallback instance.GetData() return nil")
		return &DefaultImportsHandler{}
	}

	cb, ok := v.(*abiContext)
	if !ok {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] getInstanceCallback return type is not *abiContext")
		return &DefaultImportsHandler{}
	}

	if cb.imports == nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] getInstanceCallback imports not set")
		return &DefaultImportsHandler{}
	}

	return cb.imports
}

func getHttpStruct(instance types.WasmInstance) *httpStruct {
	v := instance.GetData()
	if v == nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] getHttpStruct instance.GetData() return nil")
		return &httpStruct{}
	}

	cb, ok := v.(*abiContext)
	if !ok {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] getHttpStruct return type is not *abiImple")
		return &httpStruct{}
	}

	if cb.httpCallout == nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] getHttpStruct not have httpCallout")
		return &httpStruct{}
	}

	return cb.httpCallout
}

func GetBuffer(instance types.WasmInstance, bufferType BufferType) buffer.IoBuffer {
	switch bufferType {
	case BufferTypeHttpRequestBody:
		callback := getInstanceCallback(instance)
		return callback.GetHttpRequestBody()
	case BufferTypeHttpResponseBody:
		callback := getInstanceCallback(instance)
		return callback.GetHttpResponseBody()
	case BufferTypePluginConfiguration:
		callback := getInstanceCallback(instance)
		return callback.GetPluginConfig()
	case BufferTypeVmConfiguration:
		callback := getInstanceCallback(instance)
		return callback.GetVmConfig()
	case BufferTypeHttpCallResponseBody:
		httpStruct := getHttpStruct(instance)
		return httpStruct.responseBody
	}
	return nil
}

func GetMap(instance types.WasmInstance, mapType MapType) api.HeaderMap {
	switch mapType {
	case MapTypeHttpRequestHeaders:
		callback := getInstanceCallback(instance)
		return callback.GetHttpRequestHeader()
	case MapTypeHttpRequestTrailers:
		callback := getInstanceCallback(instance)
		return callback.GetHttpRequestTrailer()
	case MapTypeHttpResponseHeaders:
		callback := getInstanceCallback(instance)
		return callback.GetHttpResponseHeader()
	case MapTypeHttpResponseTrailers:
		callback := getInstanceCallback(instance)
		return callback.GetHttpResponseTrailer()
	case MapTypeHttpCallResponseHeaders:
		httpStruct := getHttpStruct(instance)
		return httpStruct.responseHeader
	case MapTypeHttpCallResponseTrailers:
		httpStruct := getHttpStruct(instance)
		return httpStruct.responseTrailer
	}
	return nil
}

func proxyLog(instance types.WasmInstance, level int32, logDataPtr int32, logDataSize int32) int32 {
	callback := getInstanceCallback(instance)

	logContent, err := instance.GetMemory(uint64(logDataPtr), uint64(logDataSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	callback.Log(toMosnLogLevel(LogLevel(level)), string(logContent))

	return WasmResultOk.Int32()
}

func proxyGetBufferBytes(instance types.WasmInstance, bufferType int32, start int32, length int32, returnBufferData int32, returnBufferSize int32) int32 {
	if BufferType(bufferType) > BufferTypeMax {
		return WasmResultBadArgument.Int32()
	}

	buf := GetBuffer(instance, BufferType(bufferType))
	if buf == nil {
		return WasmResultNotFound.Int32()
	}

	if start > start+length {
		return WasmResultBadArgument.Int32()
	}

	if start+length > int32(buf.Len()) {
		length = int32(buf.Len()) - start
	}

	addr, err := instance.Malloc(int32(length))
	if err != nil {
		return WasmResultInternalFailure.Int32()
	}

	err = instance.PutMemory(addr, uint64(length), buf.Bytes())
	if err != nil {
		return WasmResultInternalFailure.Int32()
	}

	_ = instance.PutUint32(uint64(returnBufferData), uint32(addr))
	_ = instance.PutUint32(uint64(returnBufferSize), uint32(length))

	return WasmResultOk.Int32()
}

func proxySetBufferBytes(instance types.WasmInstance, bufferType int32, start int32, length int32, dataPtr int32, dataSize int32) int32 {
	if BufferType(bufferType) > BufferTypeMax {
		return WasmResultBadArgument.Int32()
	}

	buf := GetBuffer(instance, BufferType(bufferType))
	if buf == nil {
		return WasmResultNotFound.Int32()
	}

	content, err := instance.GetMemory(uint64(dataPtr), uint64(dataSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	if start+length > int32(buf.Len()) {
		length = int32(buf.Len()) - start
	}

	copy(buf.Bytes()[start:start+length], content)

	return WasmResultOk.Int32()
}

func proxyGetHeaderMapPairs(instance types.WasmInstance, mapType int32, returnDataPtr int32, returnDataSize int32) int32 {
	if MapType(mapType) > MapTypeMax {
		return WasmResultBadArgument.Int32()
	}

	header := GetMap(instance, MapType(mapType))
	if header == nil {
		return WasmResultNotFound.Int32()
	}

	cloneMap := make(map[string]string)
	totalBytesLen := 4
	header.Range(func(key, value string) bool {
		cloneMap[key] = value
		totalBytesLen += 4 + 4                         // keyLen + valueLen
		totalBytesLen += len(key) + 1 + len(value) + 1 // key + \0 + value + \0
		return true
	})

	addr, err := instance.Malloc(int32(totalBytesLen))
	if err != nil {
		log.DefaultLogger.Errorf("wasm malloc error: %v", err)
		return WasmResultInternalFailure.Int32()
	}

	err = instance.PutUint32(addr, uint32(len(cloneMap)))

	lenPtr := addr + 4
	dataPtr := lenPtr + uint64(8*len(cloneMap))
	for k, v := range cloneMap {
		err = instance.PutUint32(lenPtr, uint32(len(k)))
		lenPtr += 4
		err = instance.PutUint32(lenPtr, uint32(len(v)))
		lenPtr += 4

		err = instance.PutMemory(dataPtr, uint64(len(k)), []byte(k))
		dataPtr += uint64(len(k))
		err = instance.PutByte(dataPtr, 0)
		dataPtr++

		err = instance.PutMemory(dataPtr, uint64(len(v)), []byte(v))
		dataPtr += uint64(len(v))
		err = instance.PutByte(dataPtr, 0)
		dataPtr++
	}

	err = instance.PutUint32(uint64(returnDataPtr), uint32(addr))
	err = instance.PutUint32(uint64(returnDataSize), uint32(totalBytesLen))

	return WasmResultOk.Int32()
}

func proxySetHeaderMapPairs(instance types.WasmInstance, mapType int32, ptr int32, size int32) int32 {
	if MapType(mapType) > MapTypeMax {
		return WasmResultBadArgument.Int32()
	}

	headerMap := GetMap(instance, MapType(mapType))
	if headerMap == nil {
		return WasmResultNotFound.Int32()
	}

	newMapContent, err := instance.GetMemory(uint64(ptr), uint64(size))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	newMap := DecodeMap(newMapContent)

	for k, v := range newMap {
		headerMap.Set(k, v)
	}

	return WasmResultOk.Int32()
}

func proxyGetHeaderMapValue(instance types.WasmInstance, mapType int32, keyDataPtr int32, keySize int32, valueDataPtr int32, valueSize int32) int32 {
	if MapType(mapType) > MapTypeMax {
		return WasmResultBadArgument.Int32()
	}

	headerMap := GetMap(instance, MapType(mapType))
	if headerMap == nil {
		return WasmResultNotFound.Int32()
	}

	key, err := instance.GetMemory(uint64(keyDataPtr), uint64(keySize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(key) == 0 {
		return WasmResultBadArgument.Int32()
	}

	value, ok := headerMap.Get(string(key))
	if !ok {
		return WasmResultNotFound.Int32()
	}

	addr, err := instance.Malloc(int32(len(value)))
	if err != nil {
		return WasmResultInternalFailure.Int32()
	}

	err = instance.PutMemory(addr, uint64(len(value)), []byte(value))

	err = instance.PutUint32(uint64(valueDataPtr), uint32(addr))
	err = instance.PutUint32(uint64(valueSize), uint32(len(value)))

	return WasmResultOk.Int32()
}

func proxyReplaceHeaderMapValue(instance types.WasmInstance, mapType int32, keyDataPtr int32, keySize int32, valueDataPtr int32, valueSize int32) int32 {
	if MapType(mapType) > MapTypeMax {
		return WasmResultBadArgument.Int32()
	}

	headerMap := GetMap(instance, MapType(mapType))
	if headerMap == nil {
		return WasmResultNotFound.Int32()
	}

	key, err := instance.GetMemory(uint64(keyDataPtr), uint64(keySize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(key) == 0 {
		return WasmResultBadArgument.Int32()
	}

	value, err := instance.GetMemory(uint64(valueDataPtr), uint64(valueSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(value) == 0 {
		return WasmResultBadArgument.Int32()
	}

	headerMap.Set(string(key), string(value))

	return WasmResultOk.Int32()
}

func proxyAddHeaderMapValue(instance types.WasmInstance, mapType int32, keyDataPtr int32, keySize int32, valueDataPtr int32, valueSize int32) int32 {
	if MapType(mapType) > MapTypeMax {
		return WasmResultBadArgument.Int32()
	}

	headerMap := GetMap(instance, MapType(mapType))
	if headerMap == nil {
		return WasmResultNotFound.Int32()
	}

	key, err := instance.GetMemory(uint64(keyDataPtr), uint64(keySize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(key) == 0 {
		return WasmResultBadArgument.Int32()
	}

	value, err := instance.GetMemory(uint64(valueDataPtr), uint64(valueSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(value) == 0 {
		return WasmResultBadArgument.Int32()
	}

	headerMap.Add(string(key), string(value))

	return WasmResultOk.Int32()
}

func proxyRemoveHeaderMapValue(instance types.WasmInstance, mapType int32, keyDataPtr int32, keySize int32) int32 {
	if MapType(mapType) > MapTypeMax {
		return WasmResultBadArgument.Int32()
	}

	headerMap := GetMap(instance, MapType(mapType))
	if headerMap == nil {
		return WasmResultNotFound.Int32()
	}

	key, err := instance.GetMemory(uint64(keyDataPtr), uint64(keySize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(key) == 0 {
		return WasmResultBadArgument.Int32()
	}

	headerMap.Del(string(key))

	return WasmResultOk.Int32()
}

func proxyGetProperty(instance types.WasmInstance, keyPtr int32, keySize int32, returnValueData int32, returnValueSize int32) int32 {
	return WasmResultOk.Int32()
}

func proxySetProperty(instance types.WasmInstance, keyPtr int32, keySize int32, valuePtr int32, valueSize int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxySetEffectiveContext(instance types.WasmInstance, contextID int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxySetTickPeriodMilliseconds(instance types.WasmInstance, tickPeriodMilliseconds int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyGetCurrentTimeNanoseconds(instance types.WasmInstance, resultUint64Ptr int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyGrpcCall(instance types.WasmInstance, servicePtr int32, serviceSize int32, serviceNamePtr int32, serviceNameSize int32,
	methodNamePtr int32, methodNameSize int32,
	initialMetadataPtr int32, initialMetadataSize int32,
	requestPtr int32, requestSize int32,
	timeoutMilliseconds int32, tokenPtr int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyGrpcStream(instance types.WasmInstance, servicePtr int32, serviceSize int32, serviceNamePtr int32, serviceNameSize int32,
	methodNamePtr int32, methodNameSize int32,
	initialMetadataPtr int32, initialMetadataSize int32,
	tokenPtr int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyGrpcCancel(instance types.WasmInstance, token int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyGrpcClose(instance types.WasmInstance, token int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyGrpcSend(instance types.WasmInstance, token int32, messagePtr int32, messageSize int32, endStream int32) int32 {
	return WasmResultUnimplemented.Int32()
}

type httpStruct struct {
	calloutID         int32
	instance          types.WasmInstance
	abi               *abiContext
	conn              types.ClientConnection
	connEventListener api.ConnectionEventListener
	ctx               context.Context
	streamClient      stream.Client
	receiverListener  types.StreamReceiveListener
	streamSender      types.StreamSender
	asyncRetChan      chan struct{}

	responseHeader  api.HeaderMap
	responseBody    buffer.IoBuffer
	responseTrailer api.HeaderMap
}

func (hs *httpStruct) AsyncHttpCall(timeoutMilliseconds int, u *url.URL,
	header api.HeaderMap, body buffer.IoBuffer, trailer api.HeaderMap) error {
	remoteAddr, err := net.ResolveTCPAddr("tcp", u.Host)
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] AsyncHttpCall fail to resolve tcp addr, err: %v", err)
		return err
	}

	// TODO: deal with timeout: http call timeout and dial timeout

	conn := network.NewClientConnection(time.Duration(timeoutMilliseconds)*time.Millisecond,
		nil, remoteAddr, make(chan struct{}))
	if err := conn.Connect(); err != nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] AsyncHttpCall fail to create conn, err: %v", err)
		return err
	}
	hs.conn = conn
	hs.connEventListener = &proxyHttpConnEventListener{conn: conn, httpStruct: hs}
	conn.AddConnectionEventListener(hs.connEventListener)

	hs.ctx = variable.NewVariableContext(context.Background())

	_ = variable.SetVariableValue(hs.ctx, types.VarHost, u.Host)
	_ = variable.SetVariableValue(hs.ctx, types.VarPath, u.Path)
	_ = variable.SetVariableValue(hs.ctx, types.VarMethod, http.MethodGet)
	_ = variable.SetVariableValue(hs.ctx, types.VarQueryString, u.RawQuery)

	hs.streamClient = stream.NewStreamClient(hs.ctx, protocol.HTTP1, conn, nil)
	if hs.streamClient == nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] AsyncHttpCall fail to create http stream, err: %v", err)
		return err
	}

	hs.receiverListener = &proxyStreamReceiveListener{httpStruct: hs}

	hs.streamSender = hs.streamClient.NewStream(hs.ctx, hs.receiverListener)

	notHaveData := body == nil || body.Len() == 0
	notHaveTrailer := trailer == nil

	hs.asyncRetChan = make(chan struct{}, 1)

	err = hs.streamSender.AppendHeaders(hs.ctx, header, notHaveData && notHaveTrailer)
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] AsyncHttpCall AppendHeaders fail, err: %v", err)
		return err
	}

	if !notHaveData {
		err = hs.streamSender.AppendData(hs.ctx, body, notHaveTrailer)
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] AsyncHttpCall AppendData fail, err: %v", err)
			return err
		}
	}

	if !notHaveTrailer {
		err = hs.streamSender.AppendTrailers(hs.ctx, trailer)
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] AsyncHttpCall AppendTrailers fail, err: %v", err)
			return err
		}
	}

	return nil
}

type proxyHttpConnEventListener struct {
	conn       types.ClientConnection
	httpStruct *httpStruct
}

func (p *proxyHttpConnEventListener) OnEvent(event api.ConnectionEvent) {
	log.DefaultLogger.Infof("[proxywasm_0_1_0][imports] httpCall OnEvent: %v", event)
	if p.httpStruct.asyncRetChan != nil {
		close(p.httpStruct.asyncRetChan)
		p.httpStruct.asyncRetChan = nil
	}
}

type proxyStreamReceiveListener struct {
	httpStruct *httpStruct
}

func (p *proxyStreamReceiveListener) OnReceive(ctx context.Context, headers api.HeaderMap, data buffer.IoBuffer, trailers api.HeaderMap) {
	abi := p.httpStruct.abi
	if abi == nil {
		return
	}

	defer func() {
		if p.httpStruct.asyncRetChan != nil {
			close(p.httpStruct.asyncRetChan)
			p.httpStruct.asyncRetChan = nil
		}
	}()

	rootContextID := abi.imports.GetRootContextID()

	p.httpStruct.responseHeader = headers
	p.httpStruct.responseBody = data
	p.httpStruct.responseTrailer = trailers

	err := abi.ProxyOnHttpCallResponse(rootContextID, p.httpStruct.calloutID, 0, int32(data.Len()), 0)
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] OnReceive fail to call ProxyOnHttpCallResponse, err: %v", err)
	}
}

func (p *proxyStreamReceiveListener) OnDecodeError(ctx context.Context, err error, headers api.HeaderMap) {
	log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] httpCall OnDecodeError, err: %v", err)
	if p.httpStruct.asyncRetChan != nil {
		close(p.httpStruct.asyncRetChan)
		p.httpStruct.asyncRetChan = nil
	}
}

var httpCalloutID int32 = 0

func proxyHttpCall(instance types.WasmInstance, uriPtr int32, uriSize int32,
	headerPairsPtr int32, headerPairsSize int32,
	bodyPtr int32, bodySize int32,
	trailerPairsPtr int32, trailerPairsSize int32,
	timeoutMilliseconds int32, calloutIDPtr int32) int32 {

	urlStr, err := instance.GetMemory(uint64(uriPtr), uint64(uriSize))
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] proxyHttpCall fail to get url str from instance mem, err: %v", err)
		return WasmResultInvalidMemoryAccess.Int32()
	}

	u, err := url.Parse(string(urlStr))
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] proxyHttpCall fail to parse url, err: %v, urlStr: %v", err, urlStr)
		return WasmResultBadArgument.Int32()
	}

	var header api.HeaderMap
	var body buffer.IoBuffer
	var trailer api.HeaderMap

	if headerPairsSize > 0 {
		headerStr, err := instance.GetMemory(uint64(headerPairsPtr), uint64(headerPairsSize))
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] proxyHttpCall fail to get header str from instance mem, err: %v", err)
			return WasmResultInvalidMemoryAccess.Int32()
		}
		headerMap := DecodeMap(headerStr)
		header = mosnhttp.RequestHeader{RequestHeader: &fasthttp.RequestHeader{}}
		for k, v := range headerMap {
			header.Add(k, v)
		}
	}

	if bodySize > 0 {
		bodyStr, err := instance.GetMemory(uint64(bodyPtr), uint64(bodySize))
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] proxyHttpCall fail to get body str from instance mem, err: %v", err)
			return WasmResultInvalidMemoryAccess.Int32()
		}
		body = buffer.NewIoBufferBytes(bodyStr)
	}

	if trailerPairsSize > 0 {
		trailerStr, err := instance.GetMemory(uint64(trailerPairsPtr), uint64(trailerPairsSize))
		if err != nil {
			log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] proxyHttpCall fail to get trailer str from instance mem, err: %v", err)
			return WasmResultInvalidMemoryAccess.Int32()
		}
		trailerMap := DecodeMap(trailerStr)
		if trailerMap != nil {
			trailer = mosnhttp.RequestHeader{RequestHeader: &fasthttp.RequestHeader{}}
			for k, v := range trailerMap {
				trailer.Add(k, v)
			}
		}
	}

	calloutID := atomic.AddInt32(&httpCalloutID, 1)

	hs := &httpStruct{
		calloutID: calloutID,
		instance:  instance,
		abi:       instance.GetData().(*abiContext),
	}
	hs.abi.httpCallout = hs

	err = hs.AsyncHttpCall(int(timeoutMilliseconds), u, header, body, trailer)
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] proxyHttpCall fail to launch async http call, err: %v", err)
		return WasmResultInternalFailure.Int32()
	}

	err = instance.PutUint32(uint64(calloutIDPtr), uint32(calloutID))
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm_0_1_0][imports] proxyHttpCall fail to return calloutID, err: %v", err)
		return WasmResultInvalidMemoryAccess.Int32()
	}

	return WasmResultOk.Int32()
}

func proxyDefineMetric(instance types.WasmInstance, metricType int32, namePtr int32, nameSize int32, resultPtr int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyIncrementMetric(instance types.WasmInstance, metricId int32, offset int64) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyRecordMetric(instance types.WasmInstance, metricId int32, value int64) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyGetMetric(instance types.WasmInstance, metricId int32, resultUint64Ptr int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyRegisterSharedQueue(instance types.WasmInstance, queueNamePtr int32, queueNameSize int32, tokenPtr int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyResolveSharedQueue(instance types.WasmInstance, vmIdPtr int32, vmIdSize int32, queueNamePtr int32, queueNameSize int32, tokenPtr int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyDequeueSharedQueue(instance types.WasmInstance, token int32, dataPtr int32, dataSize int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyEnqueueSharedQueue(instance types.WasmInstance, token int32, dataPtr int32, dataSize int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxyGetSharedData(instance types.WasmInstance, keyPtr int32, keySize int32, valuePtr int32, valueSizePtr int32, casPtr int32) int32 {
	return WasmResultUnimplemented.Int32()
}

func proxySetSharedData(instance types.WasmInstance, keyPtr int32, keySize int32, valuePtr int32, valueSize int32, cas int32) int32 {
	return WasmResultUnimplemented.Int32()
}
