package mhttp2

import (
	"github.com/alipay/sofa-mosn/pkg/module/http2"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func EngineServer(sc *http2.MServerConn) types.ProtocolEngine {
	return rpc.NewEngine(&serverCodec{sc: sc}, &serverCodec{sc: sc})
}

func EngineClient(cc *http2.MClientConn) types.ProtocolEngine {
	return rpc.NewEngine(&clientCodec{cc: cc}, &clientCodec{cc: cc})
}