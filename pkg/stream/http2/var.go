package http2

import (
	"context"
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

var (
	headerIndex = len(types.VarPrefixHttp2Header)
)

func init() {
	variable.RegisterPrefixVariable(types.VarPrefixHttp2Header,
		variable.NewBasicVariable(types.VarPrefixHttp2Header, nil, headerGetter, nil, 0))
	variable.RegisterPrefixVariable(types.VarPrefixHttp2Cookie,
		variable.NewBasicVariable(types.VarPrefixHttp2Cookie, nil, cookieGetter, nil, 0))

	variable.RegisterProtocolResource(protocol.HTTP2, api.HEADER, types.VarProtocolRequestHeader)
	variable.RegisterProtocolResource(protocol.HTTP2, api.COOKIE, types.VarPrefixHttpCookie)
}

func cookieGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (s string, err error) {
	return "", nil
}

func headerGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (s string, err error) {
	headers, ok := mosnctx.Get(ctx, types.ContextKeyDownStreamHeaders).(api.HeaderMap)
	if !ok {
		return variable.ValueNotFound, nil
	}
	headerKey, ok := data.(string)
	if !ok {
		return variable.ValueNotFound, nil
	}

	header, found := headers.Get(headerKey[headerIndex:])
	if !found {
		return variable.ValueNotFound, nil
	}

	return header, nil
}
