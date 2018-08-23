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
	"sync"

	"github.com/alipay/sofa-mosn/pkg/types"
)

var ioBufferPools [maxPoolSize]IoBufferPool

// IoBufferPool is Iobuffer Pool
type IoBufferPool struct {
	pool       sync.Pool
}

// NewIoBufferPool returns IoBufferPool
func NewIoBufferPool() *IoBufferPool {
	i := bufferPoolIndex()
	return &ioBufferPools[i]
}

// Take returns IoBuffer from IoBufferPool
func (p *IoBufferPool) Take(size int) (buf types.IoBuffer) {
	v := p.pool.Get()
	if v == nil {
		buf = NewIoBuffer(size)
	} else {
		buf = v.(types.IoBuffer)
		buf.Alloc(size)
	}
	return buf
}

// Give returns IoBuffer to IoBufferPool
func (p *IoBufferPool) Give(buf types.IoBuffer) {
	buf.Free()
	p.pool.Put(buf)
}