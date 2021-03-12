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
	"mosn.io/proxy-wasm-go-host/common"
)

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
	case BufferTypeHttpCallResponseBody:
		return im.GetHttpCallResponseBody()
	case BufferTypeGrpcReceiveBuffer:
		return im.GetGrpcReceiveBuffer()
	case BufferTypePluginConfiguration:
		return im.GetPluginConfig()
	case BufferTypeVmConfiguration:
		return im.GetVmConfig()
	case BufferTypeCallData:
		return im.GetFuncCallData()
	default:
		return im.GetCustomBuffer(bufferType)
	}
}

func ProxyGetBufferBytes(instance common.WasmInstance, bufferType int32, start int32, length int32, returnBufferData int32, returnBufferSize int32) int32 {
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
		return WasmResultInvalidMemoryAccess.Int32()
	}

	err = instance.PutMemory(addr, uint64(length), buf.Bytes())
	if err != nil {
		return WasmResultInternalFailure.Int32()
	}

	err = instance.PutUint32(uint64(returnBufferData), uint32(addr))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	err = instance.PutUint32(uint64(returnBufferSize), uint32(length))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	return WasmResultOk.Int32()
}

func ProxySetBufferBytes(instance common.WasmInstance, bufferType int32, start int32, length int32, dataPtr int32, dataSize int32) int32 {
	buf := GetBuffer(instance, BufferType(bufferType))
	if buf == nil {
		return WasmResultNotFound.Int32()
	}

	content, err := instance.GetMemory(uint64(dataPtr), uint64(dataSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	if start == 0 {
		if length == 0 || int(length) >= buf.Len() {
			buf.Drain(buf.Len())
			_, err = buf.Write(content)
		} else {
			return WasmResultBadArgument.Int32()
		}
	} else if int(start) >= buf.Len() {
		_, err = buf.Write(content)
	} else {
		return WasmResultBadArgument.Int32()
	}

	if err != nil {
		return WasmResultInternalFailure.Int32()
	}

	return WasmResultOk.Int32()
}
