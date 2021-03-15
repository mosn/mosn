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
	req := NewWasmRequestWithId(uint32(requestId), nil, nil)
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
	resp := NewWasmResponseWithId(uint32(request.GetRequestId()), nil, nil)
	resp.Flag |= HeartBeatFlag
	resp.ctx = wasmCtx

	wasmCtx.keepaliveResp = resp

	if !resp.IsReplacedId {
		resp.RpcId = resp.GetId()
	}

	return wasmCtx.keepaliveResp
}
