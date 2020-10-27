package pluginfilter

import (
	"context"
	"sync"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/plugin/proto"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream("pluginfilter", CreatePluginFilterFactory)
}

var client *plugin.Client
var once sync.Once

type pluginFilter struct {
	context context.Context
	// callbacks
	handler api.StreamReceiverFilterHandler
}

// NewPluginFilter used to create new plugin filter
func NewPluginFilter(context context.Context) api.StreamReceiverFilter {
	return &pluginFilter{
		context: context,
	}
}

func (f *pluginFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if client != nil {
		h := make(map[string]string)
		if headers != nil {
			headers.Range(func(k, v string) bool {
				h[k] = v
				return true
			})
		}
		t := make(map[string]string)
		if trailers != nil {
			trailers.Range(func(k, v string) bool {
				h[k] = v
				return true
			})
		}
		var body []byte
		if buf != nil {
			body = buf.Bytes()
		}
		response, err := client.Call(&proto.Request{
			Header:  h,
			Body:    body,
			Trailer: t,
		}, 10*time.Millisecond)
		if err != nil {
			log.DefaultLogger.Errorf("pluginfilter error:%v", err)
			return api.StreamFilterContinue
		}

		for k, v := range response.GetHeader() {
			if v == "" {
				headers.Del(k)
			} else {
				headers.Set(k, v)
			}
		}

		for k, v := range response.GetTrailer() {
			if v == "" {
				trailers.Del(k)
			} else {
				trailers.Set(k, v)
			}
		}

		body = response.GetBody()
		if body != nil {
			buf.Drain(buf.Len())
			buf.Write(body)
		}
	}
	return api.StreamFilterContinue
}

func (f *pluginFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *pluginFilter) OnDestroy() {}

// PluginFilterConfigFactory Filter Config Factory
type PluginFilterConfigFactory struct{}

// CreateFilterChain for create plugin filter
func (f *PluginFilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	once.Do(func() {
		var err error
		client, err = plugin.Register("pluginfilter", nil)
		if err != nil {
			log.DefaultLogger.Errorf("plugin Register error: %v", err)
		}
	})
	filter := NewPluginFilter(context)
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
}

// CreatePluginFilterFactory
func CreatePluginFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	return &PluginFilterConfigFactory{}, nil
}
