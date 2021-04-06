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
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm/abi"
	"mosn.io/mosn/pkg/wasm/abi/ext/xproxywasm020"
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

func NewWasmRpcProtocol(pw types.WasmPluginWrapper, wrapper *protocolWrapper) *wasmProtocol {
	return &wasmProtocol{
		pw:      pw,
		name:    types.ProtocolName(wrapper.config.SubProtocol),
		wrapper: wrapper,
	}
}

// =========== wasm rpc v1.0 ========
// Support for classic wasm-rpc microservice calls
type wasmProtocol struct {
	wrapper *protocolWrapper
	pw      types.WasmPluginWrapper
	name    types.ProtocolName
}

// types.Protocol
func (proto *wasmProtocol) Name() types.ProtocolName {
	return proto.name
}

func (proto *wasmProtocol) Encode(ctx context.Context, message interface{}) (types.IoBuffer, error) {
	switch frame := message.(type) {
	case *Request:
		return proto.encodeRequest(ctx, frame)
	case *Response:
		return proto.encodeResponse(ctx, frame)
	default:
		log.Proxy.Errorf(ctx, "[protocol][wasm-%s] encode with unknown command : %+v", proto.name, message)
		return nil, api.ErrUnknownType
	}
}

func (proto *wasmProtocol) Decode(ctx context.Context, buf types.IoBuffer) (interface{}, error) {
	return proto.decodeCommand(ctx, buf)
}

func (proto *wasmProtocol) Trigger(ctx context.Context, requestId uint64) api.XFrame {
	return proto.keepaliveRequest(ctx, requestId)
}

func (proto *wasmProtocol) Reply(ctx context.Context, request api.XFrame) api.XRespFrame {
	return proto.keepaliveResponse(ctx, request)
}

// Hijacker
func (proto *wasmProtocol) Hijack(ctx context.Context, request api.XFrame, statusCode uint32) api.XRespFrame {
	return proto.hijack(ctx, request, statusCode)
}

func (proto *wasmProtocol) Mapping(httpStatusCode uint32) uint32 {
	return httpStatusCode
}

// need plugin report
func (proto *wasmProtocol) PoolMode() api.PoolMode {
	return proto.wrapper.config.poolMode
}

func (proto *wasmProtocol) EnableWorkerPool() bool {
	return !proto.wrapper.config.DisableWorkerPool
}

// generate a request id for stream to combine stream request && response
// use connection param as base
func (proto *wasmProtocol) GenerateRequestID(streamID *uint64) uint64 {
	if !proto.wrapper.config.PluginGenerateID {
		return atomic.AddUint64(streamID, 1)
	}

	// todo sdk should exports abi
	return atomic.AddUint64(streamID, 1)
}

func (proto *wasmProtocol) OnProxyCreate(context context.Context) *Context {
	buf := bufferByContext(context)
	if buf.wasmCtx == nil {
		buf.wasmCtx = proto.newWasmContext(context)
	}
	return buf.wasmCtx
}

func (proto *wasmProtocol) newWasmContext(_ context.Context) *Context {
	wasmCtx := proto.NewContext()
	// save current wasm context wasmCtx
	wasmCtx.instance.Lock(wasmCtx.abi)
	wasmCtx.abi.SetABIImports(wasmCtx)
	// invoke plugin proxy on create
	err := wasmCtx.exports.ProxyOnContextCreate(wasmCtx.contextId, proto.wrapper.config.RootContextID)
	if err != nil {
		log.DefaultLogger.Warnf("failed to create protocol '%s' context, contextId %d not found", proto.name, wasmCtx.contextId)
	}
	wasmCtx.instance.Unlock()
	return wasmCtx
}

func (proto *wasmProtocol) OnProxyDelete(context context.Context) {
	buf := bufferByContext(context)
	if buf.wasmCtx == nil {
		return
	}

	wasmCtx := buf.wasmCtx
	wasmCtx.instance.Lock(wasmCtx.abi)
	wasmCtx.abi.SetABIImports(wasmCtx)
	// invoke plugin proxy on done
	wasmCtx.exports.ProxyOnDelete(wasmCtx.contextId)
	// remove wasm context
	buf.wasmCtx = nil
	wasmCtx.instance.Unlock()
}

func (proto *wasmProtocol) NewContext() *Context {
	instance := proto.pw.GetPlugin().GetInstance()
	abiVersion := abi.GetABI(instance, xproxywasm020.AbiV2)
	ctx := &Context{
		abi:       abiVersion,
		proto:     proto,
		contextId: atomic.AddInt32(&contextId, 1),
		exports:   abiVersion.(xproxywasm020.Exports),
		instance:  instance,
	}
	return ctx
}

func (proto *wasmProtocol) finishWasmContext(context context.Context) {
	proto.OnProxyDelete(context)
}
