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
	"bytes"
	"sync"
)

var (
	defaultPool *ObjectPool
)

func init() {
	defaultPool = NewObjectPool(func() PoolObject {
		return new(bytes.Buffer)
	})
}

// GetBytesBuffer returns bytes.Buffer from pool
func GetBytesBuffer() *bytes.Buffer {
	return defaultPool.Get().(*bytes.Buffer)
}

// PutIoBuffer returns IoBuffer to pool
func PutBytesBuffer(buf *bytes.Buffer) {
	defaultPool.Put(buf)
}

// Pool object
type PoolObject interface {
	Reset()
}

type New func() PoolObject

// Pool is bytes.Buffer Pool
type ObjectPool struct {
	New  New
	pool sync.Pool
}

func NewObjectPool(n New) *ObjectPool {
	return &ObjectPool{New: n}
}

// take returns *bytes.Buffer from Pool
func (p *ObjectPool) Get() PoolObject {
	v := p.pool.Get()
	if v == nil {
		return p.New()
	}

	return v.(PoolObject)
}

// give returns *byes.Buffer to Pool
func (p *ObjectPool) Put(o PoolObject) {
	o.Reset()
	p.pool.Put(o)
}
