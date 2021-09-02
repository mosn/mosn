package grpcmetric

import (
	"context"

	"mosn.io/mosn/pkg/types"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream(v2.GrpcMetricFilter, buildStream)
}

type factory struct{}

func buildStream(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	return &factory{}, nil
}

func (f *factory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := &metricFilter{}
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
	callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}

type metricFilter struct {
	handler api.StreamReceiverFilterHandler
}

func (d *metricFilter) OnDestroy() {}

func (d *metricFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	return api.StreamFilterContinue
}

func (d *metricFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	d.handler = handler
}

func (d *metricFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	service, ok := headers.Get(types.GrpcServiceName)
	if !ok {
		return api.StreamFilterContinue
	}
	result, ok := headers.Get(types.GrpcRequestResult)
	if !ok {
		return api.StreamFilterContinue
	}
	stats := getStats(service)
	if stats == nil {
		return api.StreamFilterContinue
	}
	stats.RequestServiceTootle.Inc(1)
	if result == types.SUCCESS {
		stats.ResponseSuccess.Inc(1)
	} else {
		stats.ResponseFail.Inc(1)
	}
	return api.StreamFilterContinue
}

func (d *metricFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {

}
