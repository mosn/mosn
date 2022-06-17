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

package wasm

import (
	"context"

	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/buffer"
)

var rpcBufCtx wasmRpcBufferCtx

func init() {
	buffer.RegisterBuffer(&rpcBufCtx)
}

type wasmRpcBufferCtx struct {
	buffer.TempBufferCtx
}

func (ctx wasmRpcBufferCtx) New() interface{} {
	return new(wasmRpcBuffer)
}

func (ctx wasmRpcBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*wasmRpcBuffer)

	// recycle ioBuffer
	if buf.request.Data != nil {
		if e := buffer.PutIoBuffer(buf.request.Data); e != nil {
			log.DefaultLogger.Errorf("[protocol] [wasm] [buffer] [reset] PutIoBuffer error: %v", e)
		}
	}

	if buf.response.Data != nil {
		if e := buffer.PutIoBuffer(buf.response.Data); e != nil {
			log.DefaultLogger.Errorf("[protocol] [wasm] [buffer] [reset] PutIoBuffer error: %v", e)
		}
	}

	// clean wasm plugin
	buf.request.clean()
	buf.response.clean()

	buf.wasmCtx = nil

	*buf = wasmRpcBuffer{}
}

type wasmRpcBuffer struct {
	request  Request
	response Response
	wasmCtx  *Context
}

func bufferByContext(ctx context.Context) *wasmRpcBuffer {
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(&rpcBufCtx, nil).(*wasmRpcBuffer)
}
