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
	"github.com/alipay/sofa-mosn/pkg/types"
	"runtime"
	"sync"
	"sync/atomic"
)

const maxPoolSize = 1

// Register the bufferpool's name
const (
	Protocol = iota
	SofaProtocol
	Stream
	SofaStream
	Proxy
	Bytes
	End
)

var nullBufferCtx [End]interface{}

var (
	index                   uint32
	poolSize                = runtime.NumCPU()
	bufferPoolContainers    [maxPoolSize]bufferPoolContainer
	bufferPoolCtxContainers [maxPoolSize]bufferPoolCtxContainer
)

type bufferPoolContainer struct {
	pool [End]bufferPool
}

// bufferPool is buffer pool
type bufferPool struct {
	ctx types.BufferPoolCtx
	sync.Pool
}

type bufferPoolCtxContainer struct {
	sync.Pool
}

// Take returns a buffer from buffer pool
func (p *bufferPool) take(i interface{}) (value interface{}) {
	value = p.Get()
	if value == nil {
		value = p.ctx.New(i)
	}
	return
}

// Give returns a buffer to buffer pool
func (p *bufferPool) give(value interface{}) {
	p.ctx.Reset(value)
	p.Put(value)
}

// PoolCtx is buffer pool's context
type PoolCtx struct {
	*bufferPoolContainer
	value [End]interface{}
	copy [End]interface{}
}

// NewBufferPoolContext returns a context with PoolCtx
func NewBufferPoolContext(ctx context.Context, copy bool) context.Context {
	if copy {
		bufferCtx := PoolContext(ctx)
		return context.WithValue(ctx, types.ContextKeyBufferPoolCtx, bufferCtxCopy(bufferCtx))
	}

	return context.WithValue(ctx, types.ContextKeyBufferPoolCtx, NewBufferPoolCtx())
}

// CopyBufferPoolContext copy a context
func CopyBufferPoolContext(stream context.Context, conn context.Context) {
	sctx := PoolContext(stream)
	cctx := PoolContext(conn)
	sctx.copy = cctx.value
	cctx.value = nullBufferCtx
}

func bufferPoolIndex() int {
	i := atomic.AddUint32(&index, 1)
	i = i % uint32(maxPoolSize) % uint32(poolSize)
	return int(i)
}

// NewBufferPoolCtx returns PoolCtx
func NewBufferPoolCtx() (ctx *PoolCtx) {
	i := bufferPoolIndex()
	value := bufferPoolCtxContainers[i].Get()
	if value == nil {
		ctx = &PoolCtx{
			bufferPoolContainer: &bufferPoolContainers[i],
		}
	} else {
		ctx = value.(*PoolCtx)
	}
	return ctx
}

func initBufferPoolCtx(poolCtx types.BufferPoolCtx) {
	for i := 0; i < maxPoolSize; i++ {
		pool := &bufferPoolContainers[i].pool[poolCtx.Name()]
		pool.ctx = poolCtx
	}
}

// GetPool returns buffer pool
func (ctx *PoolCtx) getPool(poolCtx types.BufferPoolCtx) *bufferPool {
	pool := &ctx.pool[poolCtx.Name()]
	if pool.ctx == nil {
		initBufferPoolCtx(poolCtx)
	}
	return pool
}

// Find returns buffer from PoolCtx
func (ctx *PoolCtx) Find(poolCtx types.BufferPoolCtx, i interface{}) (value interface{}) {
	if ctx.value[poolCtx.Name()] != nil {
		return ctx.value[poolCtx.Name()]
	}
	return ctx.Take(poolCtx, i)
}

// Take returns buffer from buffer pools
func (ctx *PoolCtx) Take(poolCtx types.BufferPoolCtx, i interface{}) (value interface{}) {
	pool := ctx.getPool(poolCtx)
	value = pool.take(i)
	ctx.value[poolCtx.Name()] = value
	return
}

// Give returns buffer to buffer pools
func (ctx *PoolCtx) Give() {
	for i := 0; i < len(ctx.value); i++ {
		value := ctx.value[i]
		if value != nil {
			ctx.pool[i].give(value)
		}
		value = ctx.copy[i]
		if value != nil {
			ctx.pool[i].give(value)
		}
	}
	ctx.copy = nullBufferCtx
	ctx.value = nullBufferCtx

	i := bufferPoolIndex()

	// Give PoolCtx to Pool
	bufferPoolCtxContainers[i].Put(ctx)
}

func bufferCtxCopy(ctx *PoolCtx) *PoolCtx {
	newctx := NewBufferPoolCtx()
	if ctx != nil {
		newctx.value = ctx.value
		ctx.value = nullBufferCtx
	}
	return newctx
}

// PoolContext returns PoolCtx by context
func PoolContext(context context.Context) *PoolCtx {
	if context != nil && context.Value(types.ContextKeyBufferPoolCtx) != nil {
		return context.Value(types.ContextKeyBufferPoolCtx).(*PoolCtx)
	}
	return NewBufferPoolCtx()
}