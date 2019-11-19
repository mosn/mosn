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
	"context"
	"testing"

	"runtime/debug"
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
	}
}

func BenchmarkByteMake(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testbyte()
	}
}

// Test IoBufferPool
var Buffer [Size]byte

func testiobufferpool() IoBuffer {
	b := GetIoBuffer(Size)
	b.Write(Buffer[:])
	return b
}

func testiobuffer() IoBuffer {
	b := NewIoBuffer(Size)
	b.Write(Buffer[:])
	return b
}

func BenchmarkIoBufferPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := testiobufferpool()
		PutIoBuffer(buf)
	}
}

func BenchmarkIoBuffer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testiobuffer()
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

//test bufferpool
var mock mock_bufferctx

type mock_bufferctx struct {
	TempBufferCtx
}

func (ctx *mock_bufferctx) New() interface{} {
	return new(mock_buffers)
}

func (ctx *mock_bufferctx) Reset(x interface{}) {
	buf := x.(*mock_buffers)
	*buf = mock_buffers{}
}

type mock_buffers struct {
	m [10]int
}

func mock_BuffersByContext(ctx context.Context) *mock_buffers {
	poolCtx := PoolContext(ctx)
	return poolCtx.Find(&mock, nil).(*mock_buffers)
}

func Test_BufferPool_Register(t *testing.T) {
	defer func() {
		if p := recover(); p != nil {
			t.Log("expected panic")
		}
	}()
	ctx1 := NewBufferPoolContext(context.Background())
	mock_BuffersByContext(ctx1)
	t.Errorf("should panic")

}

func Test_BufferPool(t *testing.T) {
	// close GC
	debug.SetGCPercent(100000)
	var null [10]int

	RegisterBuffer(&mock)

	// first
	ctx1 := NewBufferPoolContext(context.Background())
	buf1 := mock_BuffersByContext(ctx1)
	for i := 0; i < 10; i++ {
		buf1.m[i] = i
	}
	t.Log(buf1.m)
	PoolContext(ctx1).Give()
	if buf1.m != null {
		t.Errorf("test bufferPool Error: Reset() failed")
	}
	t.Log(buf1.m)
	t.Logf("%p", buf1)

	// second
	ctx2 := NewBufferPoolContext(context.Background())
	buf2 := mock_BuffersByContext(ctx2)
	t.Logf("%p", buf2)
	if buf1 != buf2 {
		t.Errorf("test bufferPool Error: Reuse failed")
	}

	debug.SetGCPercent(100)
}
