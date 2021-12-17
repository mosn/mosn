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
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm/abi/ext/xproxywasm020"
	v1 "mosn.io/mosn/pkg/wasm/abi/proxywasm010"
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v1"
)

type ContextCallback interface {
	// extension for abi 0_1_0
	proxywasm.ImportsHandler

	//DrainLength() uint32
	GetDecodeCmd() api.XFrame
	SetDecodeCmd(cmd api.XFrame)
	GetDecodeBuffer() api.IoBuffer

	GetEncodeCmd() api.XFrame
	SetEncodeBuffer(buf api.IoBuffer)
	GetEncodeBuffer() api.IoBuffer
}

var contextId int32

type Context struct {
	v1.DefaultImportsHandler
	decodeCmd        api.XFrame
	decodeBuffer     api.IoBuffer
	decodeWasmBuffer common.IoBuffer
	encodeCmd        api.XFrame
	encodeBuffer     api.IoBuffer
	encodeWasmBuffer common.IoBuffer
	proto            *wasmProtocol
	keepaliveReq     *Request
	keepaliveResp    *Response
	contextId        int32
	exports          xproxywasm020.Exports
	abi              types.ABI
	instance         types.WasmInstance
	current          context.Context
}

func (c *Context) GetDecodeCmd() api.XFrame {
	return c.decodeCmd
}

func (c *Context) SetDecodeCmd(cmd api.XFrame) {
	c.decodeCmd = cmd
}

func (c *Context) GetDecodeBuffer() api.IoBuffer {
	return c.decodeBuffer
}

func (c *Context) SetDecodeBuffer(buf api.IoBuffer) {
	c.decodeBuffer = buf
}

func (c *Context) GetEncodeCmd() api.XFrame {
	return c.encodeCmd
}

func (c *Context) SetEncodeCmd(cmd api.XFrame) {
	c.encodeCmd = cmd
}

func (c *Context) SetEncodeBuffer(buf api.IoBuffer) {
	c.encodeBuffer = buf
}

func (c *Context) GetEncodeBuffer() api.IoBuffer {
	return c.encodeBuffer
}

func (c *Context) ContextId() int32 {
	return c.contextId
}

func (c *Context) GetCustomBuffer(bufferType proxywasm.BufferType) common.IoBuffer {
	switch bufferType {
	case BufferTypeDecodeData:
		c.decodeWasmBuffer = common.NewIoBufferBytes([]byte{})
		return c.decodeWasmBuffer
	case BufferTypeEncodeData:
		c.encodeWasmBuffer = common.NewIoBufferBytes([]byte{})
		return c.encodeWasmBuffer
	default:
		return nil
	}
}
