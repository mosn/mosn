package faulttolerance

import (
	"context"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol/rpc/sofarpc"
	"mosn.io/pkg/buffer"
)

type FaultToleranceFilter struct {
	config            *v2.FaultToleranceFilterConfig
	handler           api.StreamSenderFilterHandler
	newDimension      func(api.HeaderMap) InvocationStatDimension
	invocationFactory *InvocationStatFactory
	calculatePool     *CalculatePool
}

func NewFaultToleranceFilter(config *v2.FaultToleranceFilterConfig) *FaultToleranceFilter {

	return &FaultToleranceFilter{
		config:        config,
		calculatePool: NewCalculatePool(),
	}
}

func (f *FaultToleranceFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	response, ok := headers.(*sofarpc.BoltResponse)
	if !ok {
		return api.StreamFilterContinue
	}

	dimension := f.newDimension(headers)
	stat := f.invocationFactory.GetInvocationStat(dimension)
	status := response.RespStatus()
	stat.Call(f.IsException(status))
	f.calculatePool.Regulate(dimension)
	return api.StreamFilterContinue
}

func (f *FaultToleranceFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.handler = handler
}

func (f *FaultToleranceFilter) OnDestroy() {

}

func (f *FaultToleranceFilter) IsException(uint32) bool {
	return false
}
