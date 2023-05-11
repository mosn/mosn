package main

import (
	"context"
	"encoding/json"

	"mosn.io/api"
	"mosn.io/pkg/buffer"

	"mosn.io/mosn/pkg/log"
)

func init() {
	api.RegisterStream("demo", CreateDemoFactory)
}

func CreateDemoFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	b, _ := json.Marshal(conf)
	m := make(map[string]string)
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &DemoFactory{
		config: m,
	}, nil
}

type DemoFactory struct {
	config map[string]string
}

func (f *DemoFactory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewDemoFilter(ctx, f.config)
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
}

type DemoFilter struct {
	config  map[string]string
	handler api.StreamReceiverFilterHandler
}

func NewDemoFilter(ctx context.Context, config map[string]string) *DemoFilter {
	return &DemoFilter{
		config: config,
	}
}

func (f *DemoFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	log.DefaultLogger.Infof("[demo] receive a request into demo filter")
	for k, v := range f.config {
		headers.Add(k, v)
	}
	return api.StreamFilterContinue
}

func (f *DemoFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *DemoFilter) OnDestroy() {}
