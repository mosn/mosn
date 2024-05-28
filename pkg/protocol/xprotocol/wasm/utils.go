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

package wasm

import (
	"encoding/binary"
	"unsafe"
)

// EncodeMap encode map to bytes
// encoded bytes format:
// pairs number + all key/value length + all key/value bytes
//
// eg: {"key1": "value1", "hello": "world"}
// 2(pairs number) + 4(key1 length) + 6(value1 length) => { "key1": "value1" } length
//   - 5(hello length) + 5(world length) => { "hello": "world" } length
//   - key1 bytes + nil byte + value1 bytes + nil byte => { "key1": "value1" } bytes
//   - hello bytes + nil byte + world bytes + nil byte => { "hello": "world" } bytes
func EncodeMap(pairs map[string]string) []byte {
	// pairs number
	size := 4
	for key, value := range pairs {
		// key/value's bytes + 8 bytes(key value len * 2)  +  2 bytes(nil * 2)
		size += len(key) + len(value) + 10
	}

	buf := make([]byte, size)
	// write pairs number
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(pairs)))

	var (
		// skip pairs number, prepare for key/value length
		base = 4
		// data offset, prepare for key/value bytes
		offset = 4 + len(pairs)*8
		// key length
		keyLen int
		// value length
		valueLen int
	)

	// write all key and value length
	for key, value := range pairs {
		// write key length to buffer
		keyLen = len(key)
		binary.LittleEndian.PutUint32(buf[base:base+4], uint32(keyLen))

		// write key bytes to buffer
		copy(buf[offset:], key)
		offset += keyLen + 1 /** 1 nil byte*/

		base += 4
		valueLen = len(value)
		// write value length to buffer
		binary.LittleEndian.PutUint32(buf[base:base+4], uint32(valueLen))

		copy(buf[offset:], value)
		offset += valueLen + 1 /** 1 nil byte*/

		// navigate to the next key-value pair position
		base += 4
	}

	return buf
}

// DecodeMap decode map from byte slice
// see EncodeMap for more detail.
func DecodeMap(buf []byte) map[string]string {

	// return default map
	if len(buf) == 0 {
		return map[string]string{}
	}

	// read pairs number
	headers := binary.LittleEndian.Uint32(buf[0:4])
	pairs := make(map[string]string, headers)
	count := int(headers)
	var (
		// skip pairs number, prepare for key/value length
		base = 4
		// data offset, prepare for key/value bytes
		offset = 4 + count*8
		// key length
		keyLen int
		// value length
		valueLen int
		// decoded key
		key string
		// decoded value
		value string
	)

	for i := 0; i < count; i++ {
		keyLen = int(binary.LittleEndian.Uint32(buf[base : base+4]))
		key = parseSliceString(buf[offset : offset+keyLen])
		// decode key
		offset += keyLen + 1 /** 1 nil byte*/

		valueLen = int(binary.LittleEndian.Uint32(buf[base+4 : base+8]))
		// decode value
		value = parseSliceString(buf[offset : offset+valueLen])
		offset += valueLen + 1 /** 1 nil byte*/

		pairs[key] = value

		// navigate to the next key-value pair position
		base += 8
	}

	return pairs
}

func parseSliceString(buf []byte) string {
	if len(buf) <= 0 {
		return ""
	}
	return *(*string)(unsafe.Pointer(&buf))
}
