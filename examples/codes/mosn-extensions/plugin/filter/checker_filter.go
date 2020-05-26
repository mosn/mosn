package main

import (
	"context"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/plugin/proto"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream("demo_plugin", CreateDemoFactory)
}

func CreateDemoFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	args := []string{}
	path, ok := conf["config_path"].(string)
	if ok {
		args = []string{"-c", path}
	}
	client, err := plugin.Register("checker", &plugin.Config{
		Args: args,
	})
	if err != nil {
		return nil, err
	}
	return &factory{
		client: client,
	}, nil
}

type factory struct {
	client *plugin.Client
}

func (f *factory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewDemoFilter(ctx, f.client)
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
}

type DemoFilter struct {
	client  *plugin.Client
	handler api.StreamReceiverFilterHandler
}

func NewDemoFilter(ctx context.Context, client *plugin.Client) *DemoFilter {
	return &DemoFilter{
		client: client,
	}
}

func (f *DemoFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *DemoFilter) OnDestroy() {}

func (f *DemoFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	h := make(map[string]string)
	headers.Range(func(k, v string) bool {
		h[k] = v
		return true
	})
	resp, err := f.client.Call(&proto.Request{
		Header: h,
	}, time.Second)
	if err != nil {
		f.handler.SendHijackReply(500, headers)
		return api.StreamFilterStop
	}
	log.DefaultLogger.Infof("get reposne status:%d", resp.Status)
	if resp.Status == -1 {
		f.handler.SendHijackReply(403, headers)
		return api.StreamFilterStop
	}
	return api.StreamFilterContinue
}
