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

package buffer

import (
	"github.com/alipay/sofa-mosn/pkg/types"
	"runtime"
	"testing"
)

// Test byteBufferPool
func testbytepool() *[]byte {
	b := GetBytes(10000)
	buf := *b
	for i := 0; i < 10000; i++ {
		buf[i] = 1
	}
	return b
}

func testbyte() []byte {
	buf := make([]byte, 10000)
	for i := 0; i < 10000; i++ {
		buf[i] = 1
	}
	return buf
}

func BenchmarkBytePool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := testbytepool()
		PutBytes(buf)
		if i%100 == 0 {
			runtime.GC()
		}
	}
}

func BenchmarkByte(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testbyte()
		if i%100 == 0 {
			runtime.GC()
		}
	}
}

// Test IoBufferPool

func testiobufferpool() types.IoBuffer {
	b := GetIoBuffer(1)
	b.Read([]byte{1})
	return b
}

func testiobuffer() types.IoBuffer {
	b := NewIoBuffer(1)
	b.Read([]byte{1})
	return b
}

func BenchmarkIoBufferPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := testiobufferpool()
		PutIoBuffer(buf)
		if i%100 == 0 {
			runtime.GC()
		}
	}
}

func BenchmarkIoBuffer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testiobuffer()
		if i%100 == 0 {
			runtime.GC()
		}
	}
}

// Test IoBuffer
func Test_read(t *testing.T) {
	str := "read_test"
	buffer := NewIoBufferString(str)

	b := make([]byte, 32)
	_, err := buffer.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if string(b[:len(str)]) != str {
		t.Fatal("err read content")
	}

	buffer = NewIoBufferString(str)

	b = make([]byte, 4)
	_, err = buffer.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if string(b) != "read" {
		t.Fatal("err read content")
	}
}
