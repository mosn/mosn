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
	"fmt"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"os"
)

func (proto *wasmRpcProtocol) decodeCommand(context context.Context, buf types.IoBuffer) (interface{}, error) {
	ctx := mosnctx.Get(context, types.ContextKeyWasmContext)
	if ctx == nil {
		log.DefaultLogger.Errorf("[protocol] wasm %s decode failed, wasm context not found.", proto.name)
		return nil, fmt.Errorf("wasm %s decode failed, wasm context not found", proto.name)
	}

	wasmCtx := ctx.(*Context)
	proto.instance.Acquire(wasmCtx)
	// The decoded data needs to be discarded
	wasmCtx.SetDecodeBuffer(buf)
	// invoke plugin decode impl
	err := wasmCtx.exports.ProxyDecodeBufferBytes(wasmCtx.contextId, buf)
	proto.instance.Release()

	cmd := wasmCtx.GetDecodeCmd()
	_, isReq := cmd.(*Request)
	_, isResp := cmd.(*Response)
	if cmd != nil {
		fmt.Fprintf(os.Stdout, "decode command, context id: %d, req(%v)|resp(%v)|hb(%v)  \n", contextId, isReq, isResp, cmd.IsHeartbeatFrame())
	}
	return cmd, err
}
