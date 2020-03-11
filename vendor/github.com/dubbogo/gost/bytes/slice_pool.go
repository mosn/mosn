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

package gxbytes

import (
	"sync"
)

const (
	minShift = 6
	maxShift = 18
	errSlot  = -1
)

var (
	defaultSlicePool *SlicePool
)

func init() {
	defaultSlicePool = NewSlicePool()
}

// SlicePool is []byte pools
type SlicePool struct {
	minShift int
	minSize  int
	maxSize  int

	pool []*sliceSlot
}

type sliceSlot struct {
	defaultSize int
	pool        sync.Pool
}

// newSlicePool returns SlicePool
func NewSlicePool() *SlicePool {
	p := &SlicePool{
		minShift: minShift,
		minSize:  1 << minShift,
		maxSize:  1 << maxShift,
	}
	for i := 0; i <= maxShift-minShift; i++ {
		slab := &sliceSlot{
			defaultSize: 1 << (uint)(i+minShift),
		}
		p.pool = append(p.pool, slab)
	}

	return p
}

func (p *SlicePool) slot(size int) int {
	if size > p.maxSize {
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

func newSlice(size int) []byte {
	return make([]byte, size)
}

// Get returns *[]byte from SlicePool
func (p *SlicePool) Get(size int) *[]byte {
	slot := p.slot(size)
	if slot == errSlot {
		b := newSlice(size)
		return &b
	}
	v := p.pool[slot].pool.Get()
	if v == nil {
		b := newSlice(p.pool[slot].defaultSize)
		b = b[0:size]
		return &b
	}
	b := v.(*[]byte)
	*b = (*b)[0:size]
	return b
}

// Put returns *[]byte to SlicePool
func (p *SlicePool) Put(buf *[]byte) {
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

// GetBytes returns *[]byte from SlicePool
func GetBytes(size int) *[]byte {
	return defaultSlicePool.Get(size)
}

// PutBytes Put *[]byte to SlicePool
func PutBytes(buf *[]byte) {
	defaultSlicePool.Put(buf)
}
