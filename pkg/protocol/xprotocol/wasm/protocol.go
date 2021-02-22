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
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm/abi"
	"sync/atomic"
)

/**
 * The wasm protocol is only a proxy user extension protocol,
 * and the protocol header is given here to extract a common field,
 * representing a common transport model.
 *
+------------------------------------------------------------------------------------------------------------------+
0       1                          5                             9                               13               17
+-------+----+-----+-------+-------+-------+-------+-------+-----+--------+-------+-------+-------+---+----+---+---+
| flag  |    request/response id   | 		timeout/status		 | 			header length 	 	  | payload length |
+-------+----------+-------+-------+-------+-------+-------+-----+--------+-------+-------+-------+---+----+---+---+
|                                           header attachment                                                      |
|                                           payload ...                                                            |
+------------------------------------------------------------------------------------------------------------------+

flag (bit) :

0                  1         2           3          4                       8
+------------------+---------+-----------+----------+--------+----+----+----+
|  response/request/one way  | heartbeat | readonly |       unused          |
+---------------------------------------------------------------------------+

*/

func NewWasmRpcProtocol(config *ProtocolConfig, instance types.WasmInstanceWrapper) *wasmRpcProtocol {
	return &wasmRpcProtocol{
		config:   config,
		instance: instance,
		name:     types.ProtocolName(config.ExtendConfig.SubProtocol),
	}
}

// =========== wasm rpc v1.0 ========
// Support for classic wasm-rpc microservice calls
type wasmRpcProtocol struct {
	config      *ProtocolConfig
	instance    types.WasmInstanceWrapper
	name        types.ProtocolName
	contexts    map[int32]*Context // wasm context id -> context
	idToContext map[uint64]int32   // cmd id -> wasm context id
}

// types.Protocol
func (proto *wasmRpcProtocol) Name() types.ProtocolName {
	return proto.name
}

func (proto *wasmRpcProtocol) Encode(ctx context.Context, message interface{}) (types.IoBuffer, error) {
	switch frame := message.(type) {
	case *Request:
		return proto.encodeRequest(ctx, frame)
	case *Response:
		return proto.encodeResponse(ctx, frame)
	default:
		log.Proxy.Errorf(ctx, "[protocol][wasm-%s] encode with unknown command : %+v", proto.name, message)
		return nil, xprotocol.ErrUnknownType
	}
}

func (proto *wasmRpcProtocol) Decode(ctx context.Context, buf types.IoBuffer) (interface{}, error) {
	return proto.decodeCommand(ctx, buf)
}

func (proto *wasmRpcProtocol) Trigger(ctx context.Context, requestId uint64) xprotocol.XFrame {
	return proto.keepaliveRequest(ctx, requestId)
}

func (proto *wasmRpcProtocol) Reply(ctx context.Context, request xprotocol.XFrame) xprotocol.XRespFrame {
	return proto.keepaliveResponse(ctx, request)
}

// Hijacker
func (proto *wasmRpcProtocol) Hijack(ctx context.Context, request xprotocol.XFrame, statusCode uint32) xprotocol.XRespFrame {
	return proto.hijack(ctx, request, statusCode)
}

func (proto *wasmRpcProtocol) Mapping(httpStatusCode uint32) uint32 {
	return httpStatusCode
}

// need plugin report
func (proto *wasmRpcProtocol) PoolMode() types.PoolMode {
	return proto.config.poolMode
}

func (proto *wasmRpcProtocol) EnableWorkerPool() bool {
	return !proto.config.DisableWorkerPool
}

// generate a request id for stream to combine stream request && response
// use connection param as base
func (proto *wasmRpcProtocol) GenerateRequestID(streamID *uint64) uint64 {
	if !proto.config.PluginGenerateID {
		return atomic.AddUint64(streamID, 1)
	}

	// todo sdk should exports abi
	return atomic.AddUint64(streamID, 1)
}

func (proto *wasmRpcProtocol) IsProxyWasm() bool {
	return true
}

func (proto *wasmRpcProtocol) OnProxyCreate(context context.Context) context.Context {
	ctx := mosnctx.Get(context, types.ContextKeyWasmContext)
	if ctx == nil {
		wasmCtx := proto.NewContext()
		// save current wasm context wasmCtx
		proto.instance.Acquire(wasmCtx)
		proto.contexts[wasmCtx.contextId] = wasmCtx
		// invoke plugin proxy on create
		wasmCtx.exports.ProxyOnContextCreate(wasmCtx.contextId, 0)
		proto.instance.Release()
		return mosnctx.WithValue(context, types.ContextKeyWasmContext, wasmCtx)
	}
	return context
}

func (proto *wasmRpcProtocol) OnProxyDone(context context.Context) {
	ctx := mosnctx.Get(context, types.ContextKeyWasmContext)
	if ctx == nil {
		return
	}

	wasmCtx := ctx.(*Context)
	proto.instance.Acquire(wasmCtx)
	// invoke plugin proxy on done
	wasmCtx.exports.ProxyOnDone(wasmCtx.contextId)
	// invoke plugin proxy log
	wasmCtx.exports.ProxyOnLog(wasmCtx.contextId)
	proto.instance.Release()

}

func (proto *wasmRpcProtocol) OnProxyDelete(context context.Context) {
	ctx := mosnctx.Get(context, types.ContextKeyWasmContext)
	if ctx == nil {
		return
	}

	wasmCtx := ctx.(*Context)
	proto.instance.Acquire(wasmCtx)
	// invoke plugin proxy on done
	wasmCtx.exports.ProxyOnDelete(wasmCtx.contextId)
	// remove wasm context
	delete(proto.contexts, wasmCtx.contextId)
	proto.instance.Release()
}

func (proto *wasmRpcProtocol) NewContext() *Context {
	abiVersion := abi.GetABI("proxy_abi_version_0_2_0")
	return &Context{
		proto:     proto,
		contextId: atomic.AddInt32(&contextId, 1),
		exports:   abiVersion.(Exports),
	}
}
