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

// BytesPool hold specific size []byte
type BytesPool struct {
	sizes  []int // sizes declare the cap of each slot
	slots  []sync.Pool
	length int
}

var defaultBytesPool = NewBytesPool([]int{512, 1 << 10, 4 << 10, 16 << 10, 64 << 10})

// NewBytesPool ...
func NewBytesPool(slotSize []int) *BytesPool {
	bp := &BytesPool{}
	bp.sizes = slotSize
	bp.length = len(bp.sizes)

	bp.slots = make([]sync.Pool, bp.length)
	for i, size := range bp.sizes {
		size := size
		bp.slots[i] = sync.Pool{New: func() interface{} {
			buf := make([]byte, 0, size)
			return &buf
		}}
	}
	return bp
}

// SetDefaultBytesPool change default pool options
func SetDefaultBytesPool(bp *BytesPool) {
	defaultBytesPool = bp
}

func (bp *BytesPool) findIndex(size int) int {
	for i := 0; i < bp.length; i++ {
		if bp.sizes[i] >= size {
			return i
		}
	}
	return bp.length
}

// AcquireBytes get specific make([]byte, 0, size)
func (bp *BytesPool) AcquireBytes(size int) *[]byte {
	idx := bp.findIndex(size)
	if idx >= bp.length {
		buf := make([]byte, 0, size)
		return &buf
	}

	bufp := bp.slots[idx].Get().(*[]byte)
	buf := (*bufp)[:size]
	return &buf
}

// ReleaseBytes ...
func (bp *BytesPool) ReleaseBytes(bufp *[]byte) {
	bufCap := cap(*bufp)
	idx := bp.findIndex(bufCap)
	if idx >= bp.length || bp.sizes[idx] != bufCap {
		return
	}

	bp.slots[idx].Put(bufp)
}

// AcquireBytes called by defaultBytesPool
func AcquireBytes(size int) *[]byte { return defaultBytesPool.AcquireBytes(size) }

// ReleaseBytes called by defaultBytesPool
func ReleaseBytes(bufp *[]byte) { defaultBytesPool.ReleaseBytes(bufp) }
