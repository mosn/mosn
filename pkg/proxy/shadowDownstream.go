package proxy

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

const ShadowTrafficpostfix = "-mosn-shadow"

func newShadowActiveStream(ctx context.Context, streamID string, proxy *proxy) *downStream {

	newCtx := buffer.NewBufferPoolContext(ctx, true)
	proxyBuffers := proxyBuffersByContext(newCtx)

	stream := &proxyBuffers.stream

	stream.streamID = streamID
	stream.context = newCtx

	// use listener's log
	stream.logger = log.ByContext(proxy.context)

	return stream
}

func CopyAndModRequestHeaders(header types.HeaderMap) types.HeaderMap {

	//newHeaderMap := header.CopyHeaderMap()
	newHeaderMap := &protocol.ShadowRequestHeader{
		HeaderMap: header,
	}

	if value, ok := header.Get(protocol.MosnHeaderHostKey); ok {
		value += ShadowTrafficpostfix
		newHeaderMap.ShadowHostName = value
	}

	return newHeaderMap
}
