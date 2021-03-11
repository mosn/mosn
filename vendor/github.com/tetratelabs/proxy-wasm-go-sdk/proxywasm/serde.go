// Copyright 2020 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxywasm

import (
	"encoding/binary"
	"unsafe"
)

func DeserializeMap(bs []byte) [][2]string {
	numHeaders := binary.LittleEndian.Uint32(bs[0:4])
	sizes := make([]int, numHeaders*2)
	for i := 0; i < len(sizes); i++ {
		s := 4 + i*4
		sizes[i] = int(binary.LittleEndian.Uint32(bs[s : s+4]))
	}

	var sizeIndex int
	var dataIndex = 4 * (1 + 2*int(numHeaders))
	ret := make([][2]string, numHeaders)
	for i := range ret {
		keySize := sizes[sizeIndex]
		sizeIndex++
		keyPtr := bs[dataIndex : dataIndex+keySize]
		key := *(*string)(unsafe.Pointer(&keyPtr))
		dataIndex += keySize + 1

		valueSize := sizes[sizeIndex]
		sizeIndex++
		valuePtr := bs[dataIndex : dataIndex+valueSize]
		value := *(*string)(unsafe.Pointer(&valuePtr))
		dataIndex += valueSize + 1
		ret[i] = [2]string{key, value}
	}
	return ret
}

func SerializeMap(ms [][2]string) []byte {
	size := 4
	for _, m := range ms {
		// key/value's bytes + len * 2 (8 bytes) + nil * 2 (2 bytes)
		size += len(m[0]) + len(m[1]) + 10
	}

	ret := make([]byte, size)
	binary.LittleEndian.PutUint32(ret[0:4], uint32(len(ms)))

	var base = 4
	for _, m := range ms {
		binary.LittleEndian.PutUint32(ret[base:base+4], uint32(len(m[0])))
		base += 4
		binary.LittleEndian.PutUint32(ret[base:base+4], uint32(len(m[1])))
		base += 4
	}

	for _, m := range ms {
		for i := 0; i < len(m[0]); i++ {
			ret[base] = m[0][i]
			base++
		}
		base++ // nil

		for i := 0; i < len(m[1]); i++ {
			ret[base] = m[1][i]
			base++
		}
		base++ // nil
	}
	return ret
}

func SerializePropertyPath(path []string) []byte {
	// TODO: for static paths, like upstream.address, we'd better pre-define []byte so that
	// 	we do not incur any serialization cost
	if len(path) == 0 {
		return []byte{}
	}

	var size int
	for _, p := range path {
		size += len(p) + 1
	}
	ret := make([]byte, 0, size)
	for _, p := range path {
		ret = append(ret, p...)
		ret = append(ret, 0)
	}
	ret = ret[:len(ret)-1]
	return ret
}
