package sofarpc

import (
	"context"
	"encoding/json"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

func init() {
	api.RegisterStream(v2.FaultTolerance, CreateSendFilterFactory)
}

type SendFilterFactory struct {
	config *v2.FaultToleranceFilterConfig
}

func (f *SendFilterFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewSendFilter(f.config)
	callbacks.AddStreamSenderFilter(filter)
}

func CreateSendFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	if filterConfig, err := parseConfig(conf); err != nil {
		return nil, err
	} else {
		return &SendFilterFactory{
			config: filterConfig,
		}, nil
	}
}

func parseConfig(config map[string]interface{}) (*v2.FaultToleranceFilterConfig, error) {
	filterConfig := &v2.FaultToleranceFilterConfig{}
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, filterConfig); err != nil {
		return nil, err
	}
	return filterConfig, nil
}
