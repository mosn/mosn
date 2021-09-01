package grpc

import (
	"context"

	"mosn.io/mosn/pkg/filter/network/grpc"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream(v2.GRPCStreamFilter, buildStream)
}

type factory struct{}

func buildStream(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	return &factory{}, nil
}

func (f *factory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := &grpcFilter{}
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
	callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}

type grpcFilter struct {
	handler api.StreamReceiverFilterHandler
}

func (d *grpcFilter) OnDestroy() {}

func (d *grpcFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	return api.StreamFilterContinue
}

func (d *grpcFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	d.handler = handler
}

func (d *grpcFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	service, ok := headers.Get(grpc.ServiceName)
	if !ok {
		return api.StreamFilterContinue
	}
	result, ok := headers.Get(grpc.RequestResult)
	if !ok {
		return api.StreamFilterContinue
	}
	stats := getStats(service)
	if stats == nil {
		return api.StreamFilterContinue
	}
	stats.RequestServiceTotle.Inc(1)
	if result == grpc.SUCCESS {
		stats.ResponseSuccess.Inc(1)
	} else {
		stats.ResponseFail.Inc(1)
	}
	return api.StreamFilterContinue
}

func (d *grpcFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {

}
