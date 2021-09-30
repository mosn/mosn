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
	"errors"
	"fmt"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
)

func (proto *wasmProtocol) decodeCommand(context context.Context, data types.IoBuffer) (interface{}, error) {
	buf := bufferByContext(context)
	if buf.wasmCtx == nil {
		buf.wasmCtx = proto.OnProxyCreate(context)
		if buf.wasmCtx == nil {
			log.DefaultLogger.Errorf("[protocol] wasm %s decode failed, wasm context not found.", proto.name)
			return nil, fmt.Errorf("wasm %s decode failed, wasm context not found", proto.name)
		}
	}

	wasmCtx := buf.wasmCtx
	wasmCtx.current = context
	wasmCtx.instance.Lock(wasmCtx.abi)
	wasmCtx.abi.SetABIImports(wasmCtx)
	// The decoded data needs to be discarded
	wasmCtx.SetDecodeBuffer(data)
	// invoke plugin decode impl
	err := wasmCtx.exports.ProxyDecodeBufferBytes(wasmCtx.contextId, data)
	wasmCtx.instance.Unlock()

	// check wasm plugin decode status
	if err != nil {
		log.DefaultLogger.Errorf("wasm %s decode failed, err: %v", proto.name, err)
		return nil, err
	}

	// need more data
	if wasmCtx.decodeWasmBuffer == nil {
		return nil, nil
	}

	cmd, err := decode(wasmCtx, wasmCtx.decodeWasmBuffer)
	if err != nil {
		log.DefaultLogger.Errorf("wasm %s decode frame failed, err: %v", proto.name, err)
		return nil, err
	}

	// keep Alive responses require special handling
	if resp, ok := cmd.(*Response); ok && resp.IsHeartbeatFrame() {
		resp.clean()
	}

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		_, isReq := cmd.(*Request)
		_, isResp := cmd.(*Response)
		if cmd != nil {
			log.DefaultLogger.Debugf("decode command, contextId: %d, req(%v)|resp(%v)|hb(%v), rpc id: %d \n", contextId, isReq, isResp, cmd.IsHeartbeatFrame(), cmd.GetRequestId())
		}
	}

	return cmd, err
}

func decode(ctx *Context, buffer common.IoBuffer) (api.XFrame, error) {
	// buffer format:
	// encoded header map | Flag | Id | (Timeout|GetStatus) | drain length | raw bytes length | raw bytes
	content := buffer.Bytes()

	headerBytes := binary.BigEndian.Uint32(content[0:4])
	headers := xprotocol.Header{}
	if headerBytes > 0 {
		xprotocol.DecodeHeader(content[4:], &headers)
	}

	flag := content[4+headerBytes]
	id := binary.BigEndian.Uint64(content[5+headerBytes:])

	cmdType := flag >> 6
	switch cmdType {
	case RequestType, RequestOneWayType:
		decodeWasmRequest(ctx, content, headerBytes, id, &headers, flag)
	case ResponseType:
		decodeWasmResponse(ctx, content, headerBytes, id, &headers, flag)
	default:
		log.DefaultLogger.Errorf("[wasm] failed to decode buffer, type = %s, value = %d", UnKnownRpcFlagType, flag)
		return nil, errors.New(fmt.Sprintf("failed to decode buffer, type = %s, value = %d", UnKnownRpcFlagType, flag))
	}
	return ctx.GetDecodeCmd(), nil
}

func decodeWasmRequest(ctx *Context, content []byte, headerBytes uint32, id uint64, headers *xprotocol.Header, flag byte) {

	// buffer format:
	// encoded header map | Flag | Id | (Timeout|GetStatus) | drain length | raw bytes length | raw bytes

	var (
		timeoutIndex = 13 + headerBytes
		drainIndex   = timeoutIndex + 4
		rawIndex     = drainIndex + 4
		byteIndex    = rawIndex + 4
	)

	// decode wasm request timeout
	timeout := binary.BigEndian.Uint32(content[timeoutIndex:])
	// decode buffer should drain length
	drainLen := binary.BigEndian.Uint32(content[drainIndex:])
	// content byte length
	rawBytesLen := binary.BigEndian.Uint32(content[rawIndex:])

	// create proxy wasm request
	payload := make([]byte, rawBytesLen)
	// wasm shared linear memory cannot be used here,
	// otherwise it will be  modified by other data.
	copy(payload, content[byteIndex:byteIndex+rawBytesLen])

	var cmdFlag = RpcRequestFlag
	// check heartbeat command
	if flag&HeartBeatFlag != 0 {
		cmdFlag = HeartBeatFlag
	}
	// check oneway request
	if flag&RpcOneWayRequestFlag == RpcOneWayRequestFlag {
		cmdFlag |= RpcOneWayRequestFlag
	}

	poolCmd := bufferByContext(ctx.current)
	req := poolCmd.request
	req.ctx = ctx

	req.RequestHeader = RequestHeader{
		RpcHeader: RpcHeader{
			Flag:   cmdFlag,
			Id:     id,
			Header: *headers,
		},
		Timeout: timeout,
	}

	req.payload = payload
	req.PayLoad = buffer.NewIoBufferBytes(payload)

	buf := ctx.GetDecodeBuffer()
	// if data without change, direct encode forward
	req.Data = buffer.GetIoBuffer(int(drainLen))
	req.Data.Write(buf.Bytes()[:drainLen])

	// we need to drain decode buffer
	if drainLen > 0 {
		buf.Drain(int(drainLen))
	}
	ctx.SetDecodeCmd(&req)
}

func decodeWasmResponse(ctx *Context, content []byte, headerBytes uint32, id uint64, headers *xprotocol.Header, flag byte) {
	// buffer format:
	// encoded header map | Flag | Id | (Timeout|GetStatus) | drain length | raw bytes length | raw bytes

	var (
		timeoutIndex = 13 + headerBytes
		drainIndex   = timeoutIndex + 4
		rawIndex     = drainIndex + 4
		byteIndex    = rawIndex + 4
	)

	// decode wasm response status
	status := binary.BigEndian.Uint32(content[timeoutIndex:])
	// decode buffer should drain length
	drainLen := binary.BigEndian.Uint32(content[drainIndex:])
	// content byte length
	rawBytesLen := binary.BigEndian.Uint32(content[rawIndex:])
	payload := make([]byte, rawBytesLen)
	// wasm shared linear memory cannot be used here,
	// otherwise it will be  modified by other data.
	copy(payload, content[byteIndex:byteIndex+rawBytesLen])

	var cmdFlag = RpcResponseFlag
	// check heartbeat command
	if flag&HeartBeatFlag != 0 {
		cmdFlag |= HeartBeatFlag
	}

	poolCmd := bufferByContext(ctx.current)
	resp := poolCmd.response
	resp.ctx = ctx

	resp.ResponseHeader = ResponseHeader{
		RpcHeader: RpcHeader{
			Flag:   cmdFlag,
			Id:     id,
			Header: *headers,
		},
		Status: status,
	}

	resp.payload = payload
	resp.PayLoad = buffer.NewIoBufferBytes(payload)

	buf := ctx.GetDecodeBuffer()
	// if data without change, direct encode forward
	resp.Data = buffer.GetIoBuffer(int(drainLen))
	resp.Data.Write(buf.Bytes()[:drainLen])

	// we need to drain decode buffer
	if drainLen > 0 {
		buf.Drain(int(drainLen))
	}
	ctx.SetDecodeCmd(&resp)
}
