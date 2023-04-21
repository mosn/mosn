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
	"errors"
	"fmt"

	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v1"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func (a *AbiV2Impl) ProxyDecodeBufferBytes(contextId int32, buf types.IoBuffer) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[export] ProxyDecodeBufferBytes contextID: %v", contextId)
	}

	instance := a.GetInstance()
	ff, err := instance.GetExportsFunc("proxy_decode_buffer_bytes")
	ctx := getInstanceCallback(instance).(*Context)
	if err != nil {
		return errors.New(fmt.Sprintf("fail to get export func: proxy_decode_buffer_bytes for plugin %s, err: %v", ctx.proto.name, err))
	}

	// allocate memory for plugin
	addr, err := instance.Malloc(int32(buf.Len()))
	if err != nil {
		return errors.New(fmt.Sprintf("failed to allocate memory for plugin %s, len: %d", ctx.proto.name, buf.Len()))
	}

	// copy decode buffer data to plugin
	err = instance.PutMemory(addr, uint64(buf.Len()), buf.Bytes())
	if err != nil {
		return errors.New(fmt.Sprintf("failed to copy decode buffer to plugin %s, len: %d", ctx.proto.name, buf.Len()))
	}

	// invoke decode for plugin
	resp, err := ff.Call(contextId, int32(addr), buf.Len())
	if err != nil {
		return errors.New(fmt.Sprintf("fail to invoke export func: proxy_decode_buffer_bytes for plugin %s, contextId: %d, err: %v", ctx.proto.name, contextId, err))
	}

	status := resp.(int32)
	// need more data ?
	// Optimize the idea of how much data to expect to reduce the number of method calls
	if status == StatusNeedMoreData.Int32() {
		return nil
	}

	// check invoke success
	if status != proxywasm.WasmResultOk.Int32() {
		return errors.New(fmt.Sprintf("plugin %s decode buffer failed, contextId: %d, len: %d", ctx.proto.name, ctx.ContextId(), buf.Len()))
	}

	// drain buffer and update decode command

	return nil
}

func (a *AbiV2Impl) ProxyEncodeRequestBufferBytes(contextId int32, cmd api.XFrame) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[export] ProxyEncodeRequestBufferBytes contextID: %v", contextId)
	}

	req := cmd.(*Request)

	// use req wasm instance context first.
	ctx := req.ctx
	if ctx == nil {
		ctx = getInstanceCallback(a.GetInstance()).(*Context)
	}

	ff, err := ctx.instance.GetExportsFunc("proxy_encode_buffer_bytes")
	if err != nil {
		log.DefaultLogger.Errorf("[export] fail to get export func: proxy_encode_buffer_bytes for plugin %s, err: %v", ctx.proto.name, err)
		return err
	}

	headerBytes := 0
	if req.GetHeader() != nil {
		headerBytes = xprotocol.GetHeaderEncodeLength(&req.Header)
	}

	drainLen := 0
	if req.GetData() != nil {
		drainLen = req.GetData().Len()
	}

	// encode data format:
	// encoded header map | Flag | replaceId, id | Timeout | drain length | raw dataBytes
	total := 4 + headerBytes + 1 + 8*2 + 4*2 + drainLen
	buf := buffer.NewIoBuffer(total)

	// encode header map
	buf.WriteUint32(uint32(headerBytes))
	// encoded header map
	if headerBytes > 0 {
		xprotocol.EncodeHeader(buf, &req.Header)
	}

	// should copy raw bytes
	flag := RpcRequestFlag
	if req.IsHeartbeatFrame() {
		flag |= HeartBeatFlag
	}
	if req.GetStreamType() == api.RequestOneWay {
		flag |= RpcOneWayRequestFlag
	}
	// write request flag
	buf.WriteByte(flag)

	// write replaced id
	buf.WriteUint64(req.GetRequestId())
	// write command id
	buf.WriteUint64(uint64(req.RpcId))

	// write timeout
	buf.WriteUint32(req.Timeout)

	// write drain length
	buf.WriteUint32(uint32(drainLen))
	if drainLen > 0 {
		// write raw dataBytes
		buf.Write(req.GetData().Bytes())
	}

	// allocate memory for plugin
	addr, err := ctx.instance.Malloc(int32(buf.Len()))
	if err != nil {
		return errors.New(fmt.Sprintf("failed to allocate memory for plugin %s, len: %d", ctx.proto.name, buf.Len()))
	}

	// copy decode buffer data to plugin
	err = ctx.instance.PutMemory(addr, uint64(buf.Len()), buf.Bytes())
	if err != nil {
		return errors.New(fmt.Sprintf("failed to copy encode request buffer to plugin %s, len: %d", ctx.proto.name, buf.Len()))
	}

	resp, err := ff.Call(contextId, int32(addr), buf.Len())
	if err != nil {
		return errors.New(fmt.Sprintf("fail to invoke export func: proxy_encode_buffer_bytes for plugin %s, err: %v", ctx.proto.name, err))
	}

	status := resp.(int32)

	// check invoke success
	if status != proxywasm.WasmResultOk.Int32() {
		return errors.New(fmt.Sprintf("plugin %s encode request buffer failed, contextId: %d, len: %d", ctx.proto.name, ctx.ContextId(), buf.Len()))
	}

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("encode request, contextId: %d , rawId: %d, cmdId: %d \n", contextId, req.RpcId, req.GetRequestId())
	}

	return nil
}

func (a *AbiV2Impl) ProxyEncodeResponseBufferBytes(contextId int32, cmd api.XRespFrame) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[export] ProxyEncodeResponseBufferBytes contextId: %v", contextId)
	}

	resp := cmd.(*Response)

	// use cmd wasm instance context first.
	ctx := resp.ctx
	if ctx == nil {
		ctx = getInstanceCallback(a.GetInstance()).(*Context)
	}

	ff, err := ctx.instance.GetExportsFunc("proxy_encode_buffer_bytes")
	if err != nil {
		log.DefaultLogger.Errorf("[export] fail to get export func: proxy_encode_buffer_bytes for plugin %s, err: %v", ctx.proto.name, err)
		return err
	}

	headerBytes := 0
	if cmd.GetHeader() != nil {
		headerBytes = xprotocol.GetHeaderEncodeLength(&resp.Header)
	}

	drainLen := 0
	if cmd.GetData() != nil {
		drainLen = cmd.GetData().Len()
	}

	// encode data format:
	// encoded header map | Flag | replaceId, id | Status | drain length | raw dataBytes
	total := 4 + headerBytes + 1 + 8*2 + 4*2 + drainLen
	buf := buffer.NewIoBuffer(total)

	// encode header map
	buf.WriteUint32(uint32(headerBytes))
	// encoded header map
	if headerBytes > 0 {
		xprotocol.EncodeHeader(buf, &resp.Header)
	}

	// should copy raw bytes
	flag := RpcResponseFlag
	if cmd.IsHeartbeatFrame() {
		flag |= HeartBeatFlag
	}
	// write request flag
	buf.WriteByte(flag)

	// write replaced id
	buf.WriteUint64(cmd.GetRequestId())
	// write command id
	buf.WriteUint64(uint64(resp.RpcId))

	// write timeout
	buf.WriteUint32(cmd.GetStatusCode())

	// write drain length
	buf.WriteUint32(uint32(drainLen))
	if drainLen > 0 {
		// write raw dataBytes
		buf.Write(cmd.GetData().Bytes())
	}

	// allocate memory for plugin
	addr, err := ctx.instance.Malloc(int32(buf.Len()))
	if err != nil {
		return errors.New(fmt.Sprintf("failed to allocate memory for plugin %s, len: %d", ctx.proto.name, buf.Len()))
	}

	// copy decode buffer data to plugin
	err = ctx.instance.PutMemory(addr, uint64(buf.Len()), buf.Bytes())
	if err != nil {
		return errors.New(fmt.Sprintf("failed to copy encode response buffer to plugin %s, len: %d", ctx.proto.name, buf.Len()))
	}

	r, err := ff.Call(contextId, int32(addr), buf.Len())
	if err != nil {
		return errors.New(fmt.Sprintf("fail to invoke export func: proxy_encode_buffer_bytes for plugin %s, err: %v", ctx.proto.name, err))
	}

	status := r.(int32)

	// check invoke success
	if status != proxywasm.WasmResultOk.Int32() {
		return errors.New(fmt.Sprintf("plugin %s encode response buffer failed, contextId: %d, len: %d", ctx.proto.name, ctx.ContextId(), buf.Len()))
	}

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("encode response, contextId: %d, rawId: %d, cmdId: %d \n", contextId, resp.RpcId, cmd.GetRequestId())
	}

	return nil
}

func (a *AbiV2Impl) ProxyKeepAliveBufferBytes(contextId int32, id uint64) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[export] ProxyKeepAliveBufferBytes contextId: %v", contextId)
	}

	instance := a.GetInstance()
	ctx := getInstanceCallback(instance).(*Context)
	ff, err := instance.GetExportsFunc("proxy_keepalive_buffer_bytes")
	if err != nil {
		log.DefaultLogger.Errorf("[export] fail to get export func: proxy_keepalive_buffer_bytes, err: %v", err)
		return err
	}

	// todo pass decode buffer to plugin
	status, err := ff.Call(contextId, int64(id))
	if err != nil {
		return err
	}

	// check invoke success
	if status != proxywasm.WasmResultOk.Int32() {
		return errors.New(fmt.Sprintf("plugin %s trigger keepalive request failed, contextId: %d, rpc id: %d", ctx.proto.name, ctx.ContextId(), id))
	}

	return nil
}

func (a *AbiV2Impl) ProxyReplyKeepAliveBufferBytes(contextId int32, cmd api.XFrame) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[export] ProxyReplyKeepAliveBufferBytes contextId: %v", contextId)
	}

	req := cmd.(*Request)

	// use cmd wasm instance context first.
	ctx := req.ctx
	if ctx == nil {
		ctx = getInstanceCallback(a.GetInstance()).(*Context)
	}

	ff, err := ctx.instance.GetExportsFunc("proxy_reply_keepalive_buffer_bytes")
	if err != nil {
		log.DefaultLogger.Errorf("[export] fail to get export func: proxy_reply_keepalive_buffer_bytes, err: %v", err)
		return err
	}

	status, err := ff.Call(contextId, 0, 0)
	if err != nil {
		return err
	}

	// check invoke success
	if status != proxywasm.WasmResultOk.Int32() {
		return errors.New(fmt.Sprintf("plugin %s trigger keepalive response failed, contextId: %d", ctx.proto.name, ctx.ContextId()))
	}

	return nil
}

func (a *AbiV2Impl) ProxyHijackBufferBytes(contextId int32, cmd api.XFrame, statusCode uint32) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[export] ProxyHijackBufferBytes contextId: %v", contextId)
	}

	req := cmd.(*Request)

	// use cmd wasm instance context first.
	ctx := req.ctx
	if ctx == nil {
		ctx = getInstanceCallback(a.GetInstance()).(*Context)
	}

	ff, err := ctx.instance.GetExportsFunc("proxy_hijack_buffer_bytes")
	if err != nil {
		log.DefaultLogger.Errorf("[export] fail to get export func: proxy_hijack_buffer_bytes, err: %v", err)
		return err
	}

	headerBytes := 0
	drainLen := 0
	// write full data structure
	if req.Data != nil {
		drainLen = req.Data.Len()
	}

	// encode data format:
	// encoded header map | Flag | replaceId, id | Timeout | drain length | raw dataBytes
	total := 4 + headerBytes + 1 + 8*2 + 4*2 + drainLen
	buf := buffer.NewIoBuffer(total)

	// encode header map
	buf.WriteUint32(uint32(headerBytes))

	// should copy raw bytes
	flag := RpcRequestFlag
	if req.IsHeartbeatFrame() {
		flag |= HeartBeatFlag
	}
	if req.GetStreamType() == api.RequestOneWay {
		flag |= RpcOneWayRequestFlag
	}
	// write request flag
	buf.WriteByte(flag)

	// write replaced id
	buf.WriteUint64(req.GetRequestId())
	// write command id
	// whether request is replaced with an ID or not,
	// what is returned here is the real ID(should be command response id)
	buf.WriteUint64(req.GetRequestId())

	// write timeout
	buf.WriteUint32(req.Timeout)

	// write drain length
	buf.WriteUint32(uint32(drainLen))
	if drainLen > 0 {
		// write raw dataBytes
		buf.Write(req.Data.Bytes())
	}

	// allocate memory for plugin
	addr, err := ctx.instance.Malloc(int32(buf.Len()))
	if err != nil {
		return errors.New(fmt.Sprintf("failed to allocate memory for plugin %s, len: %d", ctx.proto.name, buf.Len()))
	}

	// copy decode buffer data to plugin
	err = ctx.instance.PutMemory(addr, uint64(buf.Len()), buf.Bytes())
	if err != nil {
		return errors.New(fmt.Sprintf("failed to copy encode hijacker buffer to plugin %s, len: %d", ctx.proto.name, buf.Len()))
	}

	resp, err := ff.Call(contextId, int32(statusCode), int32(addr), buf.Len())
	if err != nil {
		return errors.New(fmt.Sprintf("fail to invoke export func: proxy_hijack_buffer_bytes for plugin %s, err: %v", ctx.proto.name, err))
	}

	status := resp.(int32)

	// check invoke success
	if status != proxywasm.WasmResultOk.Int32() {
		return errors.New(fmt.Sprintf("plugin %s hijack response failed, contextId: %d", ctx.proto.name, ctx.ContextId()))
	}

	return nil
}

func getInstanceCallback(instance types.WasmInstance) ContextCallback {
	v := instance.GetData()
	if v == nil {
		return &Context{}
	}

	cb, ok := v.(types.ABI)
	if !ok {
		log.DefaultLogger.Errorf("[wasm][imports] getInstanceCallback return type is not *AbiContext")
		return &Context{}
	}

	imports := cb.GetABIImports()
	if imports == nil {
		log.DefaultLogger.Errorf("[wasm][imports] getInstanceCallback imports not set")
		return &Context{}
	}

	if ctx, ok := imports.(ContextCallback); ok {
		return ctx
	}

	return &Context{}
}
