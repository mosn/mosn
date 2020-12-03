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

const (
	minShift = 6
	maxShift = 18
)

var (
	defaultSlicePool *SlicePool
)

func init() {
	defaultSlicePool = NewSlicePool()
}

// SlicePool is []byte pools
// Deprecated
type SlicePool struct {
	BytesPool
}

// newSlicePool returns SlicePool
// Deprecated
// instead by NewBytesPool
func NewSlicePool() *SlicePool {
	sizes := make([]int, 0, maxShift-minShift+1)
	for i := minShift; i <= maxShift; i++ {
		sizes = append(sizes, 1<<uint(i))
	}
	p := &SlicePool{
		*NewBytesPool(sizes),
	}
	return p
}

// Get returns *[]byte from SlicePool
func (p *SlicePool) Get(size int) *[]byte {
	return p.AcquireBytes(size)
}

// Put returns *[]byte to SlicePool
func (p *SlicePool) Put(bufp *[]byte) {
	p.ReleaseBytes(bufp)
}

// GetBytes returns *[]byte from SlicePool
// Deprecated
// instead by AcquireBytes
func GetBytes(size int) *[]byte {
	return defaultSlicePool.Get(size)
}

// PutBytes Put *[]byte to SlicePool
// Deprecated
// instead by ReleaseBytes
func PutBytes(buf *[]byte) {
	defaultSlicePool.Put(buf)
}
