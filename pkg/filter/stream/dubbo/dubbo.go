package dubbo

import (
	"context"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream(v2.DubboStream, buildStream)
}

type factory struct{}

func buildStream(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	f := &factory{}

	sskObj := conf[subsetKey]
	if sskObj == nil {
		return f, nil
	}

	ssk, ok := sskObj.(string)
	if !ok {
		return f, nil
	}
	podSubsetKey = ssk

	return f, nil
}

func (f *factory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewDubboFilter(ctx)
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
	callbacks.AddStreamSenderFilter(filter)
}

type dubboFilter struct {
	handler api.StreamReceiverFilterHandler
}

func NewDubboFilter(ctx context.Context) *dubboFilter {
	return &dubboFilter{}
}

func (d *dubboFilter) OnDestroy() {}

func (d *dubboFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {

	listener := mosnctx.Get(ctx, types.ContextKeyListenerName).(string)

	service, ok := headers.Get(dubbo.ServiceNameHeader)
	if !ok {
		log.DefaultLogger.Errorf("This filter {%s} just for dubbo protocol, please check your config.", v2.DubboStream)
		return api.StreamFilterStop
	}

	// adapte dubbo service to http host
	headers.Set(protocol.MosnHeaderHostKey, service)
	// because use http rule, so should add default path
	headers.Set(protocol.MosnHeaderPathKey, "/")

	method, _ := headers.Get(dubbo.MethodNameHeader)
	stats := getStats(listener, service, method)
	if stats != nil {
		stats.RequestServiceInfo.Inc(1)

		mosnctx.WithValue(ctx, types.ContextKeyRouteService, service)
		mosnctx.WithValue(ctx, types.ContextKeyRouteMethod, method)
	}

	for k, v := range types.GetPodLabels() {
		headers.Set(k, v)
	}

	return api.StreamFilterContinue
}

func (d *dubboFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	d.handler = handler
}

func (d *dubboFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	frame, ok := headers.(*dubbo.Frame)
	if !ok {
		log.DefaultLogger.Errorf("This filter {%s} just for dubbo protocol, please check your config.", v2.DubboStream)
		return api.StreamFilterStop
	}

	listener := mosnctx.Get(ctx, types.ContextKeyListenerName).(string)
	service := mosnctx.Get(ctx, types.ContextKeyRouteService).(string)
	method := mosnctx.Get(ctx, types.ContextKeyRouteMethod).(string)

	stats := getStats(listener, service, method)
	if stats == nil {
		return api.StreamFilterContinue
	}

	if frame.GetStatusCode() == dubbo.RespStatusOK {
		stats.ResponseSucc.Inc(1)
	} else {
		stats.ResponseFail.Inc(1)
	}
	return api.StreamFilterContinue
}

func (d *dubboFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {}
