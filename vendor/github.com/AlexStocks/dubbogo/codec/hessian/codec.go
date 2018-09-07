// Copyright (c) 2016 ~ 2018, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// pack/unpack fixed length variable

package hessian

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

func PackInt8(v int8, b []byte) []byte {
	return append(b, byte(v))
}

//[10].pack('N').bytes => [0, 0, 0, 10]
// func PackInt16(v int16, b []byte) []byte {
func PackInt16(v int16) []byte {
	var array [2]byte
	binary.BigEndian.PutUint16(array[:2], uint16(v))
	// return append(b, array[:2]...)
	return array[:]
}

//[10].pack('N').bytes => [0, 0, 0, 10]
// func PackUint16(v uint16, b []byte) []byte {
func PackUint16(v uint16) []byte {
	var array [2]byte
	binary.BigEndian.PutUint16(array[:2], v)
	// return append(b, array[:2]...)
	return array[:]
}

//[10].pack('N').bytes => [0, 0, 0, 10]
// func PackInt32(v int32, b []byte) []byte {
func PackInt32(v int32) []byte {
	var array [4]byte
	binary.BigEndian.PutUint32(array[:4], uint32(v))
	// return append(b, array[:4]...)
	return array[:]
}

//[10].pack('q>').bytes => [0, 0, 0, 0, 0, 0, 0, 10]
// func PackInt64(v int64, b []byte) []byte {
func PackInt64(v int64) []byte {
	var array [8]byte
	binary.BigEndian.PutUint64(array[:8], uint64(v))
	// return append(b, array[:8]...)
	return array[:]
}

//[10].pack('G').bytes => [64, 36, 0, 0, 0, 0, 0, 0]
// func PackFloat64(v float64, b []byte) []byte {
// 直接使用math库相关函数优化float64的pack/unpack
func PackFloat64(v float64) []byte {
	var array [8]byte
	binary.BigEndian.PutUint64(array[:8], math.Float64bits(v))
	// return append(b, array[:8]...)
	return array[:]
}

//(0,2).unpack('n')
func UnpackInt16(b []byte) int16 {
	var arr = b[:2]
	return int16(binary.BigEndian.Uint16(arr))
}

//(0,2).unpack('n')
func UnpackUint16(b []byte) uint16 {
	var arr = b[:2]
	return uint16(binary.BigEndian.Uint16(arr))
}

//(0,4).unpack('N')
func UnpackInt32(b []byte) int32 {
	var arr = b[:4]
	return int32(binary.BigEndian.Uint32(arr))
}

//long (0,8).unpack('q>')
func UnpackInt64(b []byte) int64 {
	var arr = b[:8]
	return int64(binary.BigEndian.Uint64(arr))
}

//Double (0,8).unpack('G)
func UnpackFloat64(b []byte) float64 {
	var arr = b[:8]
	return math.Float64frombits(binary.BigEndian.Uint64(arr))
}

//将字节数组格式化成 hex
func SprintHex(b []byte) (rs string) {
	rs = fmt.Sprintf("[]byte{")
	for _, v := range b {
		rs += fmt.Sprintf("0x%02x,", v)
	}
	rs = strings.TrimSpace(rs)
	rs += fmt.Sprintf("}\n")
	return
}
