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

package header

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"unsafe"

	"mosn.io/pkg/buffer"
)

// BytesKV key-value pair in byte slice
type BytesKV struct {
	Key   []byte
	Value []byte
}

// BytesHeader consists of multi key-value pair in byte slice formation. This could reduce the cost of []byte to string for protocol codec.
type BytesHeader struct {
	Kvs []BytesKV

	Changed bool
}

// ~ BytesHeaderMap
func (h *BytesHeader) Get(Key string) (Value string, ok bool) {
	for i, n := 0, len(h.Kvs); i < n; i++ {
		kv := &h.Kvs[i]
		if Key == string(kv.Key) {
			return string(kv.Value), true
		}
	}
	return "", false
}

func (h *BytesHeader) Set(Key string, Value string) {
	h.Changed = true

	for i, n := 0, len(h.Kvs); i < n; i++ {
		kv := &h.Kvs[i]
		if Key == string(kv.Key) {
			kv.Value = append(kv.Value[:0], Value...)
			return
		}
	}

	var kv *BytesKV
	h.Kvs, kv = allocKV(h.Kvs)
	kv.Key = append(kv.Key[:0], Key...)
	kv.Value = append(kv.Value[:0], Value...)
}

func (h *BytesHeader) Add(Key string, Value string) {
	panic("not supported")
}

func (h *BytesHeader) Del(Key string) {
	for i, n := 0, len(h.Kvs); i < n; i++ {
		kv := &h.Kvs[i]
		if Key == string(kv.Key) {
			h.Changed = true

			tmp := *kv
			copy(h.Kvs[i:], h.Kvs[i+1:])
			n--
			h.Kvs[n] = tmp
			h.Kvs = h.Kvs[:n]
			return
		}
	}
}

func (h *BytesHeader) Range(f func(Key, Value string) bool) {
	for i, n := 0, len(h.Kvs); i < n; i++ {
		kv := &h.Kvs[i]
		// false means stop iteration
		if !f(b2s(kv.Key), b2s(kv.Value)) {
			return
		}
	}
}

func (h *BytesHeader) Clone() *BytesHeader {
	n := len(h.Kvs)

	clone := &BytesHeader{
		Kvs: make([]BytesKV, n),
	}

	for i := 0; i < n; i++ {
		src := &h.Kvs[i]
		dst := &clone.Kvs[i]

		dst.Key = append(dst.Key[:0], src.Key...)
		dst.Value = append(dst.Value[:0], src.Value...)
	}

	return clone
}

func (h *BytesHeader) ByteSize() (size uint64) {
	for _, kv := range h.Kvs {
		size += uint64(len(kv.Key) + len(kv.Value))
	}
	return
}

func allocKV(h []BytesKV) ([]BytesKV, *BytesKV) {
	n := len(h)
	if cap(h) > n {
		h = h[:n+1]
	} else {
		h = append(h, BytesKV{})
	}
	return h, &h[n]
}

// b2s converts byte slice to a string without memory allocation.
// See https://groups.google.com/forum/#!msg/Golang-Nuts/ENgbUzYvCuU/90yGx7GUAgAJ .
//
// Note it may break if string and/or slice header will change
// in the future go versions.
func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// BytesHeader Codec
var (
	errInvalidLength = errors.New("invalid length -1, ignore current key value pair")
)

func GetHeaderEncodeLength(h *BytesHeader) (size int) {
	for i, n := 0, len(h.Kvs); i < n; i++ {
		size += 8 + len(h.Kvs[i].Key) + len(h.Kvs[i].Value)
	}
	return
}

func EncodeHeader(buf buffer.IoBuffer, h *BytesHeader) {
	for _, kv := range h.Kvs {
		encodeStr(buf, kv.Key)
		encodeStr(buf, kv.Value)
	}
}

func DecodeHeader(bytes []byte, h *BytesHeader) (err error) {
	totalLen := len(bytes)
	index := 0

	for index < totalLen {
		kv := BytesKV{}

		// 1. read key
		kv.Key, index, err = decodeStr(bytes, totalLen, index)
		if err != nil {
			if err == errInvalidLength {
				continue
			}
			return
		}

		// 2. read value
		kv.Value, index, err = decodeStr(bytes, totalLen, index)
		if err != nil {
			if err == errInvalidLength {
				continue
			}
			return
		}

		// 3. kv append
		h.Kvs = append(h.Kvs, kv)
	}
	return nil
}

func encodeStr(buf buffer.IoBuffer, str []byte) {
	length := len(str)

	// 1. encode str length
	buf.WriteUint32(uint32(length))

	// 2. encode str value
	buf.Write(str)
}

func decodeStr(bytes []byte, totalLen, index int) (str []byte, newIndex int, err error) {
	// 1. read str length
	length := binary.BigEndian.Uint32(bytes[index:])

	// avoid length = -1
	if length == math.MaxUint32 {
		return nil, index + 4, errInvalidLength
	}

	end := index + 4 + int(length)
	if end > totalLen {
		return nil, end, fmt.Errorf("decode bolt header failed, index %d, length %d, totalLen %d, bytes %v\n", index, length, totalLen, bytes)
	}

	// 2. read str value
	// should explicitly set capacity here,
	// or the append on this return value will cause override on the next bytes
	return bytes[index+4 : end : end], end, nil
}
