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

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type IoBufferPool struct {
	bufSize int
	pool    chan *IoBufferPoolEntry
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
	select {
	case bpe = <-p.pool:
		// swap out the underlying reader
		bpe.Io = r
	default:
		// none available.  create a new one
		bpe = &IoBufferPoolEntry{nil, r}
		bpe.Br = NewIoBuffer(p.bufSize)
	}

	return
}

func (p *IoBufferPool) Give(bpe *IoBufferPoolEntry) {
	bpe.Br.Reset()

	select {
	case p.pool <- bpe: // return to pool
	default: // discard
	}
}

func NewIoBufferPool(poolSize, bufferSize int) *IoBufferPool {
	return &IoBufferPool{
		bufSize: bufferSize,
		pool:    make(chan *IoBufferPoolEntry, poolSize),
	}
}
