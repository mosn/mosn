package faulttolerance

import (
	"context"
	"encoding/json"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

func init() {
	api.RegisterStream(v2.FaultTolerance, CreateFaultToleranceFilterFactory)
}

type FaultToleranceFilterFactory struct {
	Config *v2.FaultToleranceFilterConfig
}

func (f *FaultToleranceFilterFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewFaultToleranceFilter(f.Config)
	callbacks.AddStreamSenderFilter(filter)
}

func CreateFaultToleranceFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	filterConfig := &v2.FaultToleranceFilterConfig{}
	data, err := json.Marshal(filterConfig)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, filterConfig); err != nil {
		return nil, err
	}
	return &FaultToleranceFilterFactory{
		Config: filterConfig,
	}, nil
}
