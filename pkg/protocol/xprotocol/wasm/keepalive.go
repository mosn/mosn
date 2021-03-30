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

func (proto *wasmProtocol) keepaliveRequest(context context.Context, requestId uint64) api.XFrame {
	buf := bufferByContext(context)
	if buf.wasmCtx == nil {
		buf.wasmCtx = proto.OnProxyCreate(context)
		if buf.wasmCtx == nil {
			log.DefaultLogger.Errorf("[protocol] wasm %s keepalive request failed, wasm context not found.", proto.name)
			return nil
		}
	}

	wasmCtx := buf.wasmCtx
	wasmCtx.instance.Lock(wasmCtx.abi)
	wasmCtx.abi.SetABIImports(wasmCtx)

	// invoke plugin keepalive impl
	err := wasmCtx.exports.ProxyKeepAliveBufferBytes(wasmCtx.contextId, requestId)
	wasmCtx.instance.Unlock()

	// When encode is called, the proxy gets the correct buffer
	req := NewWasmRequestWithId(requestId, nil, nil)
	req.Flag = HeartBeatFlag
	req.ctx = wasmCtx

	wasmCtx.keepaliveReq = req

	// keepalive not supported.
	if err != nil {
		log.DefaultLogger.Errorf("[protocol] wasm %s keepalive request failed, err %v.", proto.name, err)
		return nil
	}

	// keepalive detect and support.
	if requestId == 0 {
		req.clean()
	}

	return wasmCtx.keepaliveReq
}

func (proto *wasmProtocol) keepaliveResponse(context context.Context, request api.XFrame) api.XRespFrame {
	buf := bufferByContext(context)
	if buf.wasmCtx == nil {
		buf.wasmCtx = proto.OnProxyCreate(context)
		if buf.wasmCtx == nil {
			log.DefaultLogger.Errorf("[protocol] wasm %s keepalive response failed, wasm context not found.", proto.name)
			return nil
		}
	}

	wasmCtx := buf.wasmCtx
	wasmCtx.instance.Lock(wasmCtx.abi)
	wasmCtx.abi.SetABIImports(wasmCtx)

	// invoke plugin keepalive impl
	err := wasmCtx.exports.ProxyReplyKeepAliveBufferBytes(wasmCtx.contextId, request.(*Request))
	wasmCtx.instance.Unlock()

	if err != nil {
		log.DefaultLogger.Errorf("[protocol] wasm %s keepalive response failed, err %v.", proto.name, err)
	}

	// When encode is called, the proxy gets the correct buffer
	resp := NewWasmResponseWithId(request.GetRequestId(), nil, nil)
	resp.Flag |= HeartBeatFlag
	resp.ctx = wasmCtx

	wasmCtx.keepaliveResp = resp

	if !resp.IsReplacedId {
		resp.RpcId = resp.GetId()
	}

	return wasmCtx.keepaliveResp
}
