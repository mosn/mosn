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
	"encoding/binary"

	"mosn.io/proxy-wasm-go-host/common"
)

func copyIntoInstance(instance common.WasmInstance, value string, retPtr int32, retSize int32) WasmResult {
	addr, err := instance.Malloc(int32(len(value)))
	if err != nil {
		return WasmResultInvalidMemoryAccess
	}

	err = instance.PutMemory(addr, uint64(len(value)), []byte(value))
	if err != nil {
		return WasmResultInvalidMemoryAccess
	}

	err = instance.PutUint32(uint64(retPtr), uint32(addr))
	if err != nil {
		return WasmResultInvalidMemoryAccess
	}

	err = instance.PutUint32(uint64(retSize), uint32(len(value)))
	if err != nil {
		return WasmResultInvalidMemoryAccess
	}

	return WasmResultOk
}

func getContextHandler(instance common.WasmInstance) ContextHandler {
	if v := instance.GetData(); v != nil {
		if im, ok := v.(ContextHandler); ok {
			return im
		}
	}

	return nil
}

func getImportHandler(instance common.WasmInstance) ImportsHandler {
	if ctx := getContextHandler(instance); ctx != nil {
		if im := ctx.GetImports(); im != nil {
			return im
		}
	}

	return &DefaultImportsHandler{}
}

// EncodeMap encode map into bytes.
func EncodeMap(m map[string]string) []byte {
	if len(m) == 0 {
		return nil
	}

	totalBytesLen := 4
	for k, v := range m {
		totalBytesLen += 4 + 4
		totalBytesLen += len(k) + 1 + len(v) + 1
	}

	b := make([]byte, totalBytesLen)
	binary.LittleEndian.PutUint32(b, uint32(len(m)))

	lenPtr := 4
	dataPtr := lenPtr + 8*len(m)

	for k, v := range m {
		binary.LittleEndian.PutUint32(b[lenPtr:], uint32(len(k)))
		lenPtr += 4
		binary.LittleEndian.PutUint32(b[lenPtr:], uint32(len(v)))
		lenPtr += 4

		copy(b[dataPtr:], k)
		dataPtr += len(k)
		b[dataPtr] = '0'
		dataPtr++

		copy(b[dataPtr:], v)
		dataPtr += len(v)
		b[dataPtr] = '0'
		dataPtr++
	}

	return b
}

// DecodeMap decode map from rawData.
func DecodeMap(rawData []byte) map[string]string {
	if len(rawData) < 4 {
		return nil
	}

	headerSize := binary.LittleEndian.Uint32(rawData[0:4])

	dataPtr := 4 + (4+4)*int(headerSize) // headerSize + (key1_size + value1_size) * headerSize
	if dataPtr >= len(rawData) {
		return nil
	}

	res := make(map[string]string)

	for i := 0; i < int(headerSize); i++ {
		lenIndex := 4 + (4+4)*i
		keySize := int(binary.LittleEndian.Uint32(rawData[lenIndex : lenIndex+4]))
		valueSize := int(binary.LittleEndian.Uint32(rawData[lenIndex+4 : lenIndex+8]))

		if dataPtr >= len(rawData) || dataPtr+keySize > len(rawData) {
			break
		}

		key := string(rawData[dataPtr : dataPtr+keySize])
		dataPtr += keySize
		dataPtr++ // 0

		if dataPtr >= len(rawData) || dataPtr+keySize > len(rawData) {
			break
		}

		value := string(rawData[dataPtr : dataPtr+valueSize])
		dataPtr += valueSize
		dataPtr++ // 0

		res[key] = value
	}

	return res
}
