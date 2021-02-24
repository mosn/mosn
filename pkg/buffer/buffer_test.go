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
	"runtime/debug"
	"testing"
)

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

func Test_TransmitBufferPoolContext(t *testing.T) {
	RegisterBuffer(&mock)
	var null [10]int

	// first
	ctx1 := NewBufferPoolContext(context.Background())
	buf1 := mock_BuffersByContext(ctx1)
	for i := 0; i < 10; i++ {
		buf1.m[i] = i
	}

	// second
	ctx2 := NewBufferPoolContext(context.Background())
	buf2 := mock_BuffersByContext(ctx2)
	for i := 0; i < 10; i++ {
		buf2.m[i] = i
	}

	TransmitBufferPoolContext(ctx2, ctx1)

	bv := PoolContext(ctx2)
	bv.Give()

	if buf1.m != null {
		t.Errorf("test bufferPool Error: Transmit failed")
	}

	if buf2.m != null {
		t.Errorf("test bufferPool Error: Transmit failed")
	}
}
