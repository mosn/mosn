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

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

// Hijacker
func (proto *wasmProtocol) hijack(context context.Context, request api.XFrame, statusCode uint32) api.XRespFrame {

	req := request.(*Request)
	wasmCtx := req.ctx
	if wasmCtx == nil {
		// the reason why the existing context is not reused is
		// that the wasm instance requested is still obtained,
		// but we need a new contextId to process the response
		wasmCtx = proto.newWasmContext(context)
		if wasmCtx == nil {
			log.DefaultLogger.Errorf("[protocol] wasm %s hijack failed, wasm context not found.", proto.name)
			return nil
		}
	}

	req.ctx = wasmCtx

	wasmCtx.instance.Lock(wasmCtx.abi)
	wasmCtx.abi.SetABIImports(wasmCtx)
	// invoke plugin hijack impl
	err := wasmCtx.exports.ProxyHijackBufferBytes(wasmCtx.contextId, req, statusCode)
	if err != nil {
		log.DefaultLogger.Errorf("[protocol] wasm %s hijack failed, contextId: %d, err %v.", wasmCtx.contextId, proto.name, err)
		return nil
	}

	wasmCtx.instance.Unlock()

	// When encode is called, the proxy gets the correct buffer
	wasmCtx.keepaliveResp = NewWasmResponseWithId(request.GetRequestId(), nil, nil)
	if req.ctx != nil {
		wasmCtx.keepaliveResp.ctx = req.ctx
	}

	return wasmCtx.keepaliveResp
}
