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

package http2

import (
	"context"

	"mosn.io/mosn/pkg/buffer"
)

var ins Http2BufferCtx

func init() {
	buffer.RegisterBuffer(&ins)
}

type Http2BufferCtx struct {
	buffer.TempBufferCtx
}

func (ctx Http2BufferCtx) New() interface{} {
	buffer := new(Http2Buffers)
	return buffer
}

func (ctx Http2BufferCtx) Reset(i interface{}) {
}

type Http2Buffers struct {
}

func Http2BuffersByContext(ctx context.Context) *Http2Buffers {
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(&ins, nil).(*Http2Buffers)
}
