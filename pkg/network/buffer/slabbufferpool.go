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
)

const MinShift = 3
const MaxShift = 15
const ErrSlot = -1

type SlabPool struct {
	minShift int
	minSize  int
	maxSize  int
	pool     []*IoBufferPool
}

func (p *SlabPool) Take(size int) types.IoBuffer {
	slot := p.slot(size)
	if slot == ErrSlot {
		return NewIoBuffer(size)
	}
	v := p.pool[slot].pool.Get()
	if v == nil {
		return NewIoBuffer(int(p.pool[slot].defaultSize))
	}
	return v.(*IoBuffer)
}

func (p *SlabPool) Clone(old types.IoBuffer) types.IoBuffer {
	size := old.Len()
	buf := p.Take(size)
	buf.Write(old.Bytes())
	return buf
}

func (p *SlabPool) Give(buf types.IoBuffer) {
	size := buf.Cap()
	buf.Reset()

	slot := p.slot(size)
	if slot == ErrSlot {
		return
	}

	if size != int(p.pool[slot].defaultSize) {
		return
	}

	p.pool[slot].pool.Put(buf)
}

func NewSlabPool() *SlabPool {
	p := &SlabPool{
		minShift: MinShift,
		minSize:  1 << MinShift,
		maxSize:  1 << MaxShift,
	}
	for i := 0; i <= MaxShift-MinShift; i++ {
		pool := &IoBufferPool{
			defaultSize: 1 << (uint)(i+MinShift),
		}
		p.pool = append(p.pool, pool)
	}

	return p
}

func (p *SlabPool) slot(size int) int {
	if size > p.maxSize {
		return ErrSlot
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
