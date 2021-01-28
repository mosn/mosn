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

package xprotocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"mosn.io/pkg/buffer"
)

var (
	errInvalidLength = errors.New("invalid length -1, ignore current key value pair")
)

func GetHeaderEncodeLength(h *Header) (size int) {
	for i, n := 0, len(h.Kvs); i < n; i++ {
		size += 8 + len(h.Kvs[i].Key) + len(h.Kvs[i].Value)
	}
	return
}

func EncodeHeader(buf buffer.IoBuffer, h *Header) {
	for _, kv := range h.Kvs {
		encodeStr(buf, kv.Key)
		encodeStr(buf, kv.Value)
	}
}

func DecodeHeader(bytes []byte, h *Header) (err error) {
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
