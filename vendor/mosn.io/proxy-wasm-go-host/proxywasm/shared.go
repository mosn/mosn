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

func ProxyGetProperty(instance common.WasmInstance, keyPtr int32, keySize int32, returnValueData int32, returnValueSize int32) int32 {
	key, err := instance.GetMemory(uint64(keyPtr), uint64(keySize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(key) == 0 {
		return WasmResultBadArgument.Int32()
	}

	ctx := getImportHandler(instance)

	value, res := ctx.GetProperty(string(key))
	if res != WasmResultOk {
		return res.Int32()
	}

	return copyIntoInstance(instance, value, returnValueData, returnValueSize).Int32()
}

func ProxySetProperty(instance common.WasmInstance, keyPtr int32, keySize int32, valuePtr int32, valueSize int32) int32 {
	key, err := instance.GetMemory(uint64(keyPtr), uint64(keySize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(key) == 0 {
		return WasmResultBadArgument.Int32()
	}

	value, err := instance.GetMemory(uint64(valuePtr), uint64(valueSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	ctx := getImportHandler(instance)

	return ctx.SetProperty(string(key), string(value)).Int32()
}

func ProxyRegisterSharedQueue(instance common.WasmInstance, queueNamePtr int32, queueNameSize int32, tokenPtr int32) int32 {
	queueName, err := instance.GetMemory(uint64(queueNamePtr), uint64(queueNameSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(queueName) == 0 {
		return WasmResultBadArgument.Int32()
	}

	ctx := getImportHandler(instance)

	queueID, res := ctx.RegisterSharedQueue(string(queueName))
	if res != WasmResultOk {
		return res.Int32()
	}

	err = instance.PutUint32(uint64(tokenPtr), uint32(queueID))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	return WasmResultOk.Int32()
}

func ProxyRemoveSharedQueue(instance common.WasmInstance, queueID int32) int32 {
	res := globalSharedQueueRegistry.delete(uint32(queueID))
	return res.Int32()
}

func ProxyResolveSharedQueue(instance common.WasmInstance, queueNamePtr int32, queueNameSize int32, tokenPtr int32) int32 {
	queueName, err := instance.GetMemory(uint64(queueNamePtr), uint64(queueNameSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(queueName) == 0 {
		return WasmResultBadArgument.Int32()
	}

	ctx := getImportHandler(instance)

	queueID, res := ctx.ResolveSharedQueue(string(queueName))
	if res != WasmResultOk {
		return res.Int32()
	}

	err = instance.PutUint32(uint64(tokenPtr), uint32(queueID))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	return WasmResultOk.Int32()
}

func ProxyDequeueSharedQueue(instance common.WasmInstance, token int32, dataPtr int32, dataSize int32) int32 {
	ctx := getImportHandler(instance)

	value, res := ctx.DequeueSharedQueue(uint32(token))
	if res != WasmResultOk {
		return res.Int32()
	}

	return copyIntoInstance(instance, value, dataPtr, dataSize).Int32()
}

func ProxyEnqueueSharedQueue(instance common.WasmInstance, token int32, dataPtr int32, dataSize int32) int32 {
	value, err := instance.GetMemory(uint64(dataPtr), uint64(dataSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	ctx := getImportHandler(instance)

	return ctx.EnqueueSharedQueue(uint32(token), string(value)).Int32()
}

func ProxyGetSharedData(instance common.WasmInstance, keyPtr int32, keySize int32, valuePtr int32, valueSizePtr int32, casPtr int32) int32 {
	key, err := instance.GetMemory(uint64(keyPtr), uint64(keySize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(key) == 0 {
		return WasmResultBadArgument.Int32()
	}

	ctx := getImportHandler(instance)

	v, cas, res := ctx.GetSharedData(string(key))
	if res != WasmResultOk {
		return res.Int32()
	}

	res = copyIntoInstance(instance, v, valuePtr, valueSizePtr)
	if res != WasmResultOk {
		return res.Int32()
	}

	err = instance.PutUint32(uint64(casPtr), uint32(cas))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	return WasmResultOk.Int32()
}

func ProxySetSharedData(instance common.WasmInstance, keyPtr int32, keySize int32, valuePtr int32, valueSize int32, cas int32) int32 {
	key, err := instance.GetMemory(uint64(keyPtr), uint64(keySize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}
	if len(key) == 0 {
		return WasmResultBadArgument.Int32()
	}

	value, err := instance.GetMemory(uint64(valuePtr), uint64(valueSize))
	if err != nil {
		return WasmResultInvalidMemoryAccess.Int32()
	}

	ctx := getImportHandler(instance)

	return ctx.SetSharedData(string(key), string(value), uint32(cas)).Int32()
}
