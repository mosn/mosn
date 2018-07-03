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
	"io"
	"sync"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type IoBufferPool struct {
	defaultSize uint64
	pool        sync.Pool
}

type IoBufferPoolEntry struct {
	Br types.IoBuffer
	Io io.ReadWriter
}

func (bpe *IoBufferPoolEntry) Read() (n int64, err error) {
	return bpe.Br.ReadOnce(bpe.Io)
}

func (bpe *IoBufferPoolEntry) Write() (n int64, err error) {
	return bpe.Br.WriteTo(bpe.Io)
}

func (p *IoBufferPool) Take(r io.ReadWriter) (bpe *IoBufferPoolEntry) {
	v := p.pool.Get()

	if v != nil {
		v.(*IoBufferPoolEntry).Io = r

		return v.(*IoBufferPoolEntry)
	}

	bpe = &IoBufferPoolEntry{nil, r}
	bpe.Br = NewIoBuffer(int(p.defaultSize))

	return
}

func (p *IoBufferPool) Give(bpe *IoBufferPoolEntry) {
	bpe.Io = nil
	bpe.Br.Reset()
	p.pool.Put(bpe)
}

func NewIoBufferPool(bufferSize int) *IoBufferPool {
	return &IoBufferPool{
		defaultSize: uint64(bufferSize),
	}
}
