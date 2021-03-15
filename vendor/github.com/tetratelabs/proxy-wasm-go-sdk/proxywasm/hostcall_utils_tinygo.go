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

// +build !proxytest

// since the difference of the types in SliceHeader.{Len, Cap} between tinygo and go,
// we have to have separated functions for converting bytes
// https://github.com/tinygo-org/tinygo/issues/1284

package proxywasm

import (
	"reflect"
	"unsafe"
)

func RawBytePtrToString(raw *byte, size int) string {
	return *(*string)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(raw)),
		Len:  uintptr(size),
		Cap:  uintptr(size),
	}))
}

func RawBytePtrToByteSlice(raw *byte, size int) []byte {
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(raw)),
		Len:  uintptr(size),
		Cap:  uintptr(size),
	}))
}
