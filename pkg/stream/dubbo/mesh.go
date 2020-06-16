package dubbo

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream("dubbo_adapter", CreateAdapter)
}

func CreateAdapter(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	return &factory{}, nil
}

type factory struct{}

func (f *factory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewDubboFilter(ctx)
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
}

type dubboFilter struct {
	handler api.StreamReceiverFilterHandler
}

func NewDubboFilter(ctx context.Context) *dubboFilter {
	return &dubboFilter{}
}

func (d *dubboFilter) OnDestroy() {}

func (d *dubboFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	service, ok := headers.Get("service")
	if ok {
		headers.Set(protocol.MosnHeaderHostKey, service)
	}

	headers.Set(protocol.MosnHeaderPathKey, "/")

	return api.StreamFilterContinue
}

func (d *dubboFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
}
