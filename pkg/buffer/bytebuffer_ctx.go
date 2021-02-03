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

	"mosn.io/pkg/buffer"
)

var ins = ByteBufferCtx{}

func init() {
	RegisterBuffer(&ins)
}

type ByteBufferCtx struct {
	TempBufferCtx
}

func (ctx ByteBufferCtx) New() interface{} {
	return buffer.NewByteBufferPoolContainer()
}

func (ctx ByteBufferCtx) Reset(i interface{}) {
	p := i.(*buffer.ByteBufferPoolContainer)
	p.Reset()
}

// GetBytesByContext returns []byte from byteBufferPool by context
func GetBytesByContext(context context.Context, size int) *[]byte {
	p := PoolContext(context).Find(&ins, nil).(*buffer.ByteBufferPoolContainer)
	return p.Take(size)
}
