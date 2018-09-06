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
	"sync"
)

const minShift = 6
const maxShift = 15
const errSlot = -1

var byteBufferPools [maxPoolSize]*byteBufferPool

func init() {
	initByterBufferPool()
}

// byteBufferPool is []byte pools
type byteBufferPool struct {
	minShift int
	minSize  int
	maxSize  int

	pool []*bufferSlot
}

type bufferSlot struct {
	defaultSize int
	pool        sync.Pool
}

// newByteBufferPool returns byteBufferPool
func newByteBufferPool() *byteBufferPool {
	p := &byteBufferPool{
		minShift: minShift,
		minSize:  1 << minShift,
		maxSize:  1 << maxShift,
	}
	for i := 0; i <= maxShift-minShift; i++ {
		slab := &bufferSlot{
			defaultSize: 1 << (uint)(i+minShift),
		}
		p.pool = append(p.pool, slab)
	}

	return p
}

func (p *byteBufferPool) slot(size int) int {
	if size > p.maxSize || size <= p.minSize {
		return errSlot
	}
	slot := 0
	shift := 0
	if size > p.minSize {
		size--
		for size > 0 {
			size = size >> 1
			shift++
		}
		slot = shift - p.minShift
	}

	return slot
}

func newBytes(size int) []byte {
	return make([]byte, size)
}

// take returns *[]byte from byteBufferPool
func (p *byteBufferPool) take(size int) *[]byte {
	slot := p.slot(size)
	if slot == errSlot {
		b := newBytes(size)
		return &b
	}
	v := p.pool[slot].pool.Get()
	if v == nil {
		b := newBytes(p.pool[slot].defaultSize)
		b = b[0:size]
		return &b
	}
	b := v.(*[]byte)
	*b = (*b)[0:size]
	return b
}

// give returns *[]byte to byteBufferPool
func (p *byteBufferPool) give(buf *[]byte) {
	if buf == nil {
		return
	}
	size := cap(*buf)
	slot := p.slot(size)
	if slot == errSlot {
		return
	}
	if size != int(p.pool[slot].defaultSize) {
		return
	}
	p.pool[slot].pool.Put(buf)
}

func initByterBufferPool() {
	for i := 0; i < maxPoolSize; i++ {
		byteBufferPools[i] = newByteBufferPool()
	}
}

// getByteBufferPool returns byteBufferPool from byteBufferPools
func getByteBufferPool() *byteBufferPool {
	i := bufferPoolIndex()
	return byteBufferPools[i]
}

type ByteBufferCtx struct{}

type ByteBufferPoolContainer struct {
	bytes []*[]byte
	*byteBufferPool
}

func (ctx ByteBufferCtx) Name() int {
	return Bytes
}

func (ctx ByteBufferCtx) New() interface{} {
	return &ByteBufferPoolContainer{
		byteBufferPool: getByteBufferPool(),
	}
}

func (ctx ByteBufferCtx) Reset(i interface{}) {
	p := i.(*ByteBufferPoolContainer)
	for _, buf := range p.bytes {
		p.give(buf)
	}
	p.bytes = p.bytes[:0]
}

// GetBytesByContext returns []byte from byteBufferPool by context
func GetBytesByContext(context context.Context, size int) *[]byte {
	p := PoolContext(context).Find(ByteBufferCtx{}, nil).(*ByteBufferPoolContainer)
	buf := p.take(size)
	p.bytes = append(p.bytes, buf)
	return buf
}

// GetBytes returns *[]byte from byteBufferPool
func GetBytes(size int) *[]byte {
	p := getByteBufferPool()
	return p.take(size)
}

// PutBytes Put *[]byte to byteBufferPool
func PutBytes(buf *[]byte) {
	p := getByteBufferPool()
	p.give(buf)
}
