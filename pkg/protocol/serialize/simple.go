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

package serialize

import (
	"encoding/binary"
	"fmt"
	"github.com/alipay/sofa-mosn/pkg/types"
	"reflect"
	"unsafe"
)

// Instance
// singleton of simpleSerialization
var Instance = simpleSerialization{}

type simpleSerialization struct{}

func (s *simpleSerialization) GetSerialNum() int {
	return 6
}

func (s *simpleSerialization) SerializeMap(m map[string]string, b types.IoBuffer) error {
	lenBuf := make([]byte, 4)

	for key, value := range m {

		keyBytes := UnsafeStrToByte(key)
		keyLen := len(keyBytes)

		binary.BigEndian.PutUint32(lenBuf, uint32(keyLen))
		b.Write(lenBuf)
		b.Write(keyBytes)

		valueBytes := UnsafeStrToByte(value)
		valueLen := len(valueBytes)

		binary.BigEndian.PutUint32(lenBuf, uint32(valueLen))
		b.Write(lenBuf)
		b.Write(valueBytes)
	}

	return nil
}

func (s *simpleSerialization) DeserializeMap(b []byte, m map[string]string) error {
	totalLen := len(b)
	index := 0

	for index < totalLen {

		length := binary.BigEndian.Uint32(b[index:])
		index += 4
		end := index + int(length)

		if end > totalLen {
			return fmt.Errorf("index %d, length %d, totalLen %d, b %v\n", index, length, totalLen, b)
		}

		key := b[index:end]
		index = end

		length = binary.BigEndian.Uint32(b[index:])
		index += 4
		end = index + int(length)

		if end > totalLen {
			return fmt.Errorf("index %d, length %d, totalLen %d, b %v\n", index, length, totalLen, b)
		}

		value := b[index:end]
		index = end

		m[string(key)] = string(value)
	}
	return nil
}

func UnsafeStrToByte(s string) []byte {
	var b []byte
	byteHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))

	// we need to take the address of s's Data field and assign it to b's Data field in one
	// expression as it as a uintptr and in the future Go may have a compacting GC that moves
	// pointers but it will not update uintptr values, but single expressions should be safe.
	// For more details see https://groups.google.com/forum/#!msg/golang-dev/rd8XgvAmtAA/p6r28fbF1QwJ
	byteHeader.Data = (*reflect.StringHeader)(unsafe.Pointer(&s)).Data

	// need to take the length of s here to ensure s is live until after we update b's Data
	// field since the garbage collector can collect a variable once it is no longer used
	// not when it goes out of scope, for more details see https://github.com/golang/go/issues/9046
	l := len(s)
	byteHeader.Len = l
	byteHeader.Cap = l

	return b
}
