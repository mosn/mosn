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

func GetMap(instance common.WasmInstance, mapType MapType) common.HeaderMap {
	ctx := getImportHandler(instance)

	switch mapType {
	case MapTypeHttpRequestHeaders:
		return ctx.GetHttpRequestHeader()
	case MapTypeHttpRequestTrailers:
		return ctx.GetHttpRequestTrailer()
	case MapTypeHttpResponseHeaders:
		return ctx.GetHttpResponseHeader()
	case MapTypeHttpResponseTrailers:
		return ctx.GetHttpResponseTrailer()
	case MapTypeGrpcReceiveInitialMetadata:
		return ctx.GetGrpcReceiveInitialMetaData()
	case MapTypeGrpcReceiveTrailingMetadata:
		return ctx.GetGrpcReceiveTrailerMetaData()
	case MapTypeHttpCallResponseHeaders:
		return ctx.GetHttpCallResponseHeaders()
	case MapTypeHttpCallResponseTrailers:
		return ctx.GetHttpCallResponseTrailer()
	default:
		return ctx.GetCustomHeader(mapType)
	}
}

func ProxyGetHeaderMapPairs(instance common.WasmInstance, mapType int32, returnDataPtr int32, returnDataSize int32) int32 {
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
		return WasmResultInvalidMemoryAccess.Int32()
	}

	err = instance.PutUint32(addr, uint32(len(cloneMap)))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
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

	err = instance.PutUint32(uint64(returnDataPtr), uint32(addr))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	err = instance.PutUint32(uint64(returnDataSize), uint32(totalBytesLen))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	return WasmResultOk.Int32()
}

func ProxySetHeaderMapPairs(instance common.WasmInstance, mapType int32, ptr int32, size int32) int32 {
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

func ProxyGetHeaderMapValue(instance common.WasmInstance, mapType int32, keyDataPtr int32, keySize int32, valueDataPtr int32, valueSize int32) int32 {
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

	return copyIntoInstance(instance, value, valueDataPtr, valueSize).Int32()
}

func ProxyReplaceHeaderMapValue(instance common.WasmInstance, mapType int32, keyDataPtr int32, keySize int32, valueDataPtr int32, valueSize int32) int32 {
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

func ProxyAddHeaderMapValue(instance common.WasmInstance, mapType int32, keyDataPtr int32, keySize int32, valueDataPtr int32, valueSize int32) int32 {
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

	headerMap.Set(string(key), string(value))

	return WasmResultOk.Int32()
}

func ProxyRemoveHeaderMapValue(instance common.WasmInstance, mapType int32, keyDataPtr int32, keySize int32) int32 {
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
