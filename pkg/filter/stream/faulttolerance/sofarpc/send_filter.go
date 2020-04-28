package sofarpc

import (
	"context"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/invocation"
	"mosn.io/mosn/pkg/protocol/rpc/sofarpc"
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

	if _, ok := headers.(*sofarpc.BoltResponse); !ok {
		if _, ok := headers.(*sofarpc.BoltRequest); !ok {
			return api.StreamFilterContinue
		}
	}

	newDimension := invocation.GetNewDimensionFunc()
	if newDimension == nil {
		return api.StreamFilterContinue
	}

	hostInfo := f.handler.RequestInfo()
	dimension := newDimension(hostInfo)
	stat := f.invocationFactory.GetInvocationStat(hostInfo.UpstreamHost(), dimension)
	config := fault_tolerance_rule.GetFaultToleranceStoreInstance().GetRule(app)
	stat.Call(config.IsException(status))

	return api.StreamFilterContinue
}

func (f *SendFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.handler = handler
}

func (f *SendFilter) OnDestroy() {

}
