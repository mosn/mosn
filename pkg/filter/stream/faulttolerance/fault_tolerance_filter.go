package faulttolerance

import (
	"context"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/pkg/buffer"
)

type FaultToleranceFilter struct {
	config        *v2.FaultToleranceFilterConfig
	newDimension  func(api.HeaderMap) InvocationStatDimension
	calculatePool *CalculatePool
}

func NewFaultToleranceFilter(config *v2.FaultToleranceFilterConfig) *FaultToleranceFilter {

	return &FaultToleranceFilter{
		config:        config,
		calculatePool: NewCalculatePool(),
	}
}

func (f *FaultToleranceFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	dimension := f.newDimension(headers)

	f.calculatePool.Regulate(headers)
}

func (f *FaultToleranceFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
}

func (f *FaultToleranceFilter) OnDestroy() {

}
