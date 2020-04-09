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
	invocationFactory *InvocationStatFactory
	hostStatusManager *HostStatusManager
}

func NewFaultToleranceFilter(config *v2.FaultToleranceFilterConfig) *FaultToleranceFilter {
	return &FaultToleranceFilter{
		config: config,
	}
}

func (f *FaultToleranceFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	response, ok := headers.(*sofarpc.BoltResponse)
	if !ok {
		return api.StreamFilterContinue
	}

	address := f.handler.RequestInfo().UpstreamHost().AddressString()
	if f.hostStatusManager.IsUnHealthy(address) {
		//f.handler.RequestInfo().UpstreamHost()
	}

	if ok, dimension := f.getInvocationDimension(response); ok {
		stat := f.invocationFactory.GetInvocationStat(dimension)
		if stat.Call(f.IsException(response.RespStatus()), f.config) {
			f.hostStatusManager.PutUnHealthyHost(dimension.dimension, address, f.GetMaxHostThreshold())
			//f.handler.RequestInfo().UpstreamHost()
		}
	}

	f.handler.RequestInfo().UpstreamHost()
	return api.StreamFilterContinue
}

func (f *FaultToleranceFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.handler = handler
}

func (f *FaultToleranceFilter) GetMaxHostThreshold() uint64 {
	result := f.config.MaxHostCount
	for _, function := range GetExtensionGetMaxHostThresholdFunc() {
		temp := function(f.config)
		if temp < result {
			result = temp
		}
	}
	return result
}

func (f *FaultToleranceFilter) OnDestroy() {

}

func (f *FaultToleranceFilter) IsException(uint32) bool {
	return false
}

func (f *FaultToleranceFilter) getInvocationDimension(headers api.HeaderMap) (bool, InvocationDimension) {
	dimensionKey := f.config.DimensionKey
	if dimension, ok := headers.Get(dimensionKey); ok {
		if requestInfo := f.handler.RequestInfo(); requestInfo != nil {
			if host := requestInfo.UpstreamHost(); host != nil {
				if address := host.AddressString(); address != "" {
					invocationDimension := NewInvocationDimension(dimension, address)
					return true, invocationDimension
				}
			}
		}
	}
	return false, GetEmptyInvocationDimension()
}
