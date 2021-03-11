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

func ProxyOpenGrpcStream(instance common.WasmInstance, grpcServiceData int32, grpcServiceSize int32,
	serviceNameData int32, serviceNameSize int32, methodData int32, methodSize int32, returnCalloutID int32) int32 {

	grpcService, err := instance.GetMemory(uint64(grpcServiceData), uint64(grpcServiceSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	serviceName, err := instance.GetMemory(uint64(serviceNameData), uint64(serviceNameSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	method, err := instance.GetMemory(uint64(methodData), uint64(methodSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	ctx := getImportHandler(instance)

	calloutID, res := ctx.OpenGrpcStream(string(grpcService), string(serviceName), string(method))
	if res != WasmResultOk {
		return res.Int32()
	}

	err = instance.PutUint32(uint64(returnCalloutID), uint32(calloutID))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	return WasmResultOk.Int32()
}

func ProxySendGrpcCallMessage(instance common.WasmInstance, calloutID int32, data int32, size int32, endOfStream int32) int32 {
	msg, err := instance.GetMemory(uint64(data), uint64(size))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	ctx := getImportHandler(instance)

	return ctx.SendGrpcCallMsg(calloutID, common.NewIoBufferBytes(msg), endOfStream).Int32()
}

func ProxyCancelGrpcCall(instance common.WasmInstance, calloutID int32) int32 {
	ctx := getImportHandler(instance)
	return ctx.CancelGrpcCall(calloutID).Int32()
}

func ProxyCloseGrpcCall(instance common.WasmInstance, calloutID int32) int32 {
	ctx := getImportHandler(instance)
	return ctx.CloseGrpcCall(calloutID).Int32()
}

func ProxyGrpcCall(instance common.WasmInstance, grpcServiceData int32, grpcServiceSize int32,
	serviceNameData int32, serviceNameSize int32,
	methodData int32, methodSize int32,
	grpcMessageData int32, grpcMessageSize int32,
	timeoutMilliseconds int32, returnCalloutID int32) int32 {

	grpcService, err := instance.GetMemory(uint64(grpcServiceData), uint64(grpcServiceSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	serviceName, err := instance.GetMemory(uint64(serviceNameData), uint64(serviceNameSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	method, err := instance.GetMemory(uint64(methodData), uint64(methodSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	msg, err := instance.GetMemory(uint64(grpcMessageData), uint64(grpcMessageSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	ctx := getImportHandler(instance)

	calloutID, res := ctx.GrpcCall(string(grpcService), string(serviceName), string(method),
		common.NewIoBufferBytes(msg), timeoutMilliseconds)
	if res != WasmResultOk {
		return res.Int32()
	}

	err = instance.PutUint32(uint64(returnCalloutID), uint32(calloutID))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	return WasmResultOk.Int32()
}
