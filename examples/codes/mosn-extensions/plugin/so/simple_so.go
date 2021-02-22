package main

import (
	"context"
	"encoding/json"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/log"
)

// define a function named: CreateFilterFactory, do not need init to register
func CreateFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	b, _ := json.Marshal(conf)
	m := make(map[string]string)
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &DemoFactory{
		config: m,
	}, nil
}

// An implementation of api.StreamFilterChainFactory
type DemoFactory struct {
	config map[string]string
}

func (f *DemoFactory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewDemoFilter(ctx, f.config)
	// ReceiverFilter, run the filter when receive a request from downstream
	// The FilterPhase can be BeforeRoute or AfterRoute, we use BeforeRoute in this demo
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
	// SenderFilter, run the filter when receive a response from upstream
	// In the demo, we are not implement this filter type
	// callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}

// What DemoFilter do:
// the request will be passed only if the request headers contains key&value matched in the config
type DemoFilter struct {
	config  map[string]string
	handler api.StreamReceiverFilterHandler
}

// NewDemoFilter returns a DemoFilter, the DemoFilter is an implementation of api.StreamReceiverFilter
// A Filter can implement both api.StreamReceiverFilter and api.StreamSenderFilter.
func NewDemoFilter(ctx context.Context, config map[string]string) *DemoFilter {
	return &DemoFilter{
		config: config,
	}
}

func (f *DemoFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	log.DefaultContextLogger.Infof(ctx, "receive a request into demo filter")
	passed := true
CHECK:
	for k, v := range f.config {
		value, ok := headers.Get(k)
		if !ok || value != v {
			passed = false
			break CHECK
		}
	}
	if !passed {
		log.DefaultContextLogger.Warnf(ctx, "request does not matched the pass condition")
		f.handler.SendHijackReply(403, headers)
		return api.StreamFilterStop
	}
	return api.StreamFilterContinue
}

func (f *DemoFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *DemoFilter) OnDestroy() {}
