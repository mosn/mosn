package grpcmetric

import (
	"context"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/filter/network/grpc"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream(v2.GrpcMetricFilter, buildStream)
}

type factory struct {
	s *state
}

func buildStream(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	return &factory{s: newState()}, nil
}

func (f *factory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := &metricFilter{ft: f}
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
	callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}

type metricFilter struct {
	handler api.StreamReceiverFilterHandler
	ft      *factory
}

func (d *metricFilter) OnDestroy() {}

func (d *metricFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	return api.StreamFilterContinue
}

func (d *metricFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	d.handler = handler
}

func (d *metricFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	service, err := variable.Get(ctx, grpc.GrpcServiceName)
	if err != nil {
		return api.StreamFilterContinue
	}
	reqResult, err := variable.Get(ctx, grpc.GrpcRequestResult)
	if err != nil {
		return api.StreamFilterContinue
	}
	costTime, err := variable.Get(ctx, grpc.GrpcServiceCostTime)
	if err != nil {
		return api.StreamFilterContinue
	}
	svcName := service.(string)
	success := reqResult.(bool)
	costTimeNs := costTime.(int64)
	stats := d.ft.s.getStats(svcName)
	if stats == nil {
		return api.StreamFilterContinue
	}
	stats.costTime.Update(costTimeNs)
	stats.requestServiceTootle.Inc(1)
	if success {
		stats.responseSuccess.Inc(1)
	} else {
		stats.responseFail.Inc(1)
	}
	return api.StreamFilterContinue
}

func (d *metricFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {

}
