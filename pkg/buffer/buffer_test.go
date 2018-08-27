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
	"runtime"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/types"
)

const Size = 2048

// Test byteBufferPool
func testbytepool() *[]byte {
	b := GetBytes(Size)
	buf := *b
	for i := 0; i < Size; i++ {
		buf[i] = 1
	}
	return b
}

func testbyte() []byte {
	buf := make([]byte, Size)
	for i := 0; i < Size; i++ {
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

func BenchmarkByteMake(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testbyte()
		if i%100 == 0 {
			runtime.GC()
		}
	}
}

// Test IoBufferPool
var Buffer [Size]byte

func testiobufferpool() types.IoBuffer {
	b := GetIoBuffer(Size)
	b.Write(Buffer[:])
	return b
}

func testiobuffer() types.IoBuffer {
	b := NewIoBuffer(Size)
	b.Write(Buffer[:])
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

func Test_IoBufferPool(t *testing.T) {
	str := "IoBufferPool Test"
	buffer := GetIoBuffer(len(str))
	buffer.Write([]byte(str))

	b := make([]byte, 32)
	_, err := buffer.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	PutIoBuffer(buffer)

	if string(b[:len(str)]) != str {
		t.Fatal("IoBufferPool Test Failed")
	}
	t.Log("IoBufferPool Test Sucess")
}

func Test_IoBufferPool_Slice_Increase(t *testing.T) {
	str := "IoBufferPool Test"
	// []byte slice increase
	buffer := GetIoBuffer(1)
	buffer.Write([]byte(str))

	b := make([]byte, 32)
	_, err := buffer.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	PutIoBuffer(buffer)

	if string(b[:len(str)]) != str {
		t.Fatal("IoBufferPool Test Slice Increase Failed")
	}
	t.Log("IoBufferPool Test Slice Increase Sucess")
}

func Test_IoBufferPool_Alloc_Free(t *testing.T) {
	str := "IoBufferPool Test"
	buffer := GetIoBuffer(100)
	buffer.Free()
	buffer.Alloc(1)
	buffer.Write([]byte(str))

	b := make([]byte, 32)
	_, err := buffer.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	PutIoBuffer(buffer)

	if string(b[:len(str)]) != str {
		t.Fatal("IoBufferPool Test Alloc Free Failed")
	}
	t.Log("IoBufferPool Test Alloc Free Sucess")
}

func Test_ByteBufferPool(t *testing.T) {
	str := "ByteBufferPool Test"
	b := GetBytes(len(str))
	buf := *b
	copy(buf, str)

	if string(buf) != str {
		t.Fatal("ByteBufferPool Test Failed")
	}
	PutBytes(b)

	b = GetBytes(len(str))
	buf = *b
	copy(buf, str)

	if string(buf) != str {
		t.Fatal("ByteBufferPool Test Failed")
	}
	PutBytes(b)
	t.Log("ByteBufferPool Test Sucess")
}
