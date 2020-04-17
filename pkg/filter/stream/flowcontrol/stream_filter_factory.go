package flowcontrol

import (
	"context"
	"encoding/json"

	"mosn.io/mosn/pkg/types"

	"github.com/alibaba/sentinel-golang/core/flow"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

const defaultResponse = "current request is limited"

// Config represents the flow control configurations.
type Config struct {
	GlobalSwitch bool                     `json:"global_switch"`
	Monitor      bool                     `json:"monitor"`
	KeyType      api.ProtocolResourceName `json:"limit_key_type"`
	Action       Action                   `json:"action"`
	Rules        []*flow.FlowRule         `json:"rules"`
}

// Action represents the direct response of request after limited.
type Action struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

func init() {
	api.RegisterStream(FlowControlFilterName, createRpcFlowControlFilterFactory)
}

// StreamFilterFactory represents the stream filter factory.
type StreamFilterFactory struct {
	config *Config
}

// CreateFilterChain add the flow control stream filter to filter chain.
func (f *StreamFilterFactory) CreateFilterChain(context context.Context,
	callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewStreamFilter(&DefaultCallbacks{config: f.config})
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
}

func createRpcFlowControlFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	flowControlCfg := &Config{
		Action: Action{
			Status: types.LimitExceededCode,
			Body:   defaultResponse,
		},
		KeyType: api.PATH,
	}
	cfg, err := json.Marshal(conf)
	if err != nil {
		log.DefaultLogger.Errorf("marshal flow control filter config failed")
		return nil, err
	}
	err = json.Unmarshal(cfg, &flowControlCfg)
	if err != nil {
		log.DefaultLogger.Errorf("parse flow control filter config failed")
		return nil, err
	}
	_, err = flow.LoadRules(flowControlCfg.Rules)
	if err != nil {
		log.DefaultLogger.Errorf("update rules failed")
		return nil, err
	}
	factory := &StreamFilterFactory{config: flowControlCfg}
	return factory, nil
}
