package wasm

import (
	"context"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func (proto *wasmRpcProtocol) keepaliveRequest(context context.Context, requestId uint64) xprotocol.XFrame {
	ctx := mosnctx.Get(context, types.ContextKeyWasmContext)
	if ctx == nil {
		log.DefaultLogger.Errorf("[protocol] wasm %s keepalive request failed, wasm context not found.", proto.name)
		return nil
	}

	wasmCtx := ctx.(*Context)
	proto.instance.Acquire(wasmCtx)
	defer proto.instance.Release()

	// invoke plugin keepalive impl
	err := wasmCtx.exports.ProxyKeepAliveBufferBytes(wasmCtx.contextId, requestId)
	if err != nil {
		log.DefaultLogger.Errorf("[protocol] wasm %s keepalive request failed, err %v.", proto.name, err)
		return nil
	}

	// todo the mock heartbeat packets
	// When encode is called, the proxy gets the correct buffer
	wasmCtx.keepaliveReq = NewWasmRequestWithId(uint32(requestId), nil, nil)
	wasmCtx.keepaliveReq.Flag = HeartBeatFlag

	return wasmCtx.keepaliveReq
}

func (proto *wasmRpcProtocol) keepaliveResponse(context context.Context, request xprotocol.XFrame) xprotocol.XRespFrame {
	ctx := mosnctx.Get(context, types.ContextKeyWasmContext)
	if ctx == nil {
		log.DefaultLogger.Errorf("[protocol] wasm %s keepalive response failed, wasm context not found.", proto.name)
		return nil
	}

	wasmCtx := ctx.(*Context)
	proto.instance.Acquire(wasmCtx)
	defer proto.instance.Release()

	// invoke plugin keepalive impl
	err := wasmCtx.exports.ProxyReplyKeepAliveBufferBytes(wasmCtx.contextId, request.(*Request))
	if err != nil {
		log.DefaultLogger.Errorf("[protocol] wasm %s keepalive response failed, err %v.", proto.name, err)
		return nil
	}

	// todo the mock heartbeat packets
	// When encode is called, the proxy gets the correct buffer
	resp := NewWasmResponseWithId(uint32(request.GetRequestId()), nil, nil)
	resp.Flag = resp.Flag | HeartBeatFlag
	wasmCtx.keepaliveResp = resp

	if !resp.IsReplacedId {
		resp.RpcId = resp.GetId()
	}

	return wasmCtx.keepaliveResp
}
