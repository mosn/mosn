package wasm

import (
	"context"

	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// Hijacker
func (proto *wasmProtocol) hijack(context context.Context, request api.XFrame, statusCode uint32) api.XRespFrame {
	ctx := mosnctx.Get(context, types.ContextKeyWasmContext)
	if ctx == nil {
		log.DefaultLogger.Errorf("[protocol] wasm %s hijack failed, wasm context not found.", proto.name)
		return nil
	}

	wasmCtx := ctx.(*Context)
	wasmCtx.instance.Lock(wasmCtx.abi)
	wasmCtx.abi.SetABIImports(wasmCtx)
	// invoke plugin hijack impl
	err := wasmCtx.exports.ProxyHijackBufferBytes(wasmCtx.contextId, request.(*Request), statusCode)
	if err != nil {
		log.DefaultLogger.Errorf("[protocol] wasm %s hijack failed, err %v.", proto.name, err)
	}

	// todo the mock heartbeat packets
	// When encode is called, the proxy gets the correct buffer
	wasmCtx.keepaliveResp = NewWasmResponseWithId(uint32(request.GetRequestId()), nil, nil)

	wasmCtx.instance.Unlock()

	return wasmCtx.keepaliveResp
}
