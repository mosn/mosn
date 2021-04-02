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
	"encoding/binary"
	"fmt"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func (proto *wasmProtocol) encodeRequest(context context.Context, request *Request) (types.IoBuffer, error) {
	wasmCtx := request.ctx
	if wasmCtx == nil {
		wasmCtx = proto.OnProxyCreate(context)
		if wasmCtx == nil {
			log.DefaultLogger.Errorf("[protocol] wasm %s encode request failed, wasm context not found.", proto.name)
			return nil, fmt.Errorf("wasm %s encode request failed, wasm context not found", proto.name)
		}
	}

	wasmCtx.instance.Lock(wasmCtx.abi)
	wasmCtx.abi.SetABIImports(wasmCtx)
	// only for debug
	wasmCtx.SetEncodeCmd(request)
	// invoke plugin encode impl
	err := wasmCtx.exports.ProxyEncodeRequestBufferBytes(wasmCtx.contextId, request)
	wasmCtx.instance.Unlock()

	// check wasm plugin encode status
	if err != nil {
		log.DefaultLogger.Errorf("wasm %s encode request failed, err: %v", proto.name, err)
		return nil, err
	}

	// encode request
	content := wasmCtx.encodeWasmBuffer.Bytes()
	encode(wasmCtx, content)

	// clean plugin context
	request.clean()

	request.ctx = nil // help gc

	return wasmCtx.GetEncodeBuffer(), err
}

func (proto *wasmProtocol) encodeResponse(context context.Context, response *Response) (types.IoBuffer, error) {
	wasmCtx := response.ctx
	if wasmCtx == nil {
		wasmCtx = proto.OnProxyCreate(context)
		if wasmCtx == nil {
			log.DefaultLogger.Errorf("[protocol] wasm %s encode response failed, wasm context not found.", proto.name)
			return nil, fmt.Errorf("wasm %s encode response failed, wasm context not found", proto.name)
		}
	}

	wasmCtx.instance.Lock(wasmCtx.abi)
	wasmCtx.abi.SetABIImports(wasmCtx)
	// only for debug
	wasmCtx.SetEncodeCmd(response)
	// invoke plugin encode impl
	err := wasmCtx.exports.ProxyEncodeResponseBufferBytes(wasmCtx.contextId, response)
	wasmCtx.instance.Unlock()

	// check wasm plugin encode status
	if err != nil {
		log.DefaultLogger.Errorf("wasm %s encode response failed, err: %v", proto.name, err)
		return nil, err
	}

	// encode request
	content := wasmCtx.encodeWasmBuffer.Bytes()
	encode(wasmCtx, content)

	// clean plugin context
	response.clean()

	response.ctx = nil // help gc

	return wasmCtx.GetEncodeBuffer(), err
}

func encode(ctx *Context, content []byte) {
	// buffer format:
	// encoded header map | Flag | Id | (Timeout|GetStatus) | drain length | raw bytes

	headerBytes := binary.BigEndian.Uint32(content[0:4])
	headers := xprotocol.Header{}
	if headerBytes > 0 {
		xprotocol.DecodeHeader(content[4:], &headers)
	}

	var (
		timeoutIndex = 13 + headerBytes
		drainIndex   = timeoutIndex + 4
		byteIndex    = drainIndex + 4
	)

	// encoded buffer length
	drainLen := binary.BigEndian.Uint32(content[drainIndex:])
	// command encode buffer
	payload := make([]byte, drainLen)
	// wasm shared linear memory cannot be used here,
	// otherwise it will be  modified by other data.
	copy(payload, content[byteIndex:byteIndex+drainLen])
	buf := buffer.NewIoBufferBytes(payload)

	ctx.SetEncodeBuffer(buf)
}
