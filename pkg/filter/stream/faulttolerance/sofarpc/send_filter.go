package sofarpc

import (
	"context"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/invocation"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/pkg/buffer"
)

type SendFilter struct {
	config            *v2.FaultToleranceFilterConfig
	invocationFactory *invocation.InvocationStatFactory
	handler           api.StreamSenderFilterHandler
}

func NewSendFilter(config *v2.FaultToleranceFilterConfig) *SendFilter {
	filter := &SendFilter{
		config: config,
	}
	return filter
}

func (f *SendFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if !f.config.Enabled {
		return api.StreamFilterContinue
	}

	if _, ok := headers.(*bolt.ResponseHeader); !ok {
		if _, ok := headers.(*bolt.RequestHeader); !ok {
			return api.StreamFilterContinue
		}
	}

	newDimension := invocation.GetNewDimensionFunc()
	if newDimension == nil {
		return api.StreamFilterContinue
	}

	requestInfo := f.handler.RequestInfo()
	host := requestInfo.UpstreamHost()
	dimension := newDimension(requestInfo)
	isException := f.IsException(requestInfo)
	stat := f.invocationFactory.GetInvocationStat(&host, dimension)
	stat.Call(isException)

	return api.StreamFilterContinue
}

func (f *SendFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.handler = handler
}

func (f *SendFilter) OnDestroy() {

}

func (f *SendFilter) IsException(requestInfo api.RequestInfo) bool {
	responseCode := requestInfo.ResponseCode()
	if _, ok := f.config.ExceptionTypes[uint32(responseCode)]; ok {
		return true
	}
	return false
}
