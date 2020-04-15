package flowcontrol

import (
	"context"
	"encoding/json"

	"github.com/alibaba/sentinel-golang/core/flow"
	"mosn.io/mosn/pkg/log"

	"mosn.io/api"
)

const alipayLimitFilterName = "alipayLimitFilter"

type FlowControlConfig struct {
	GlobalSwitch bool       `json:"global_switch"`
	Monitor      bool       `json:"monitor"`
	Rules        []FlowRule `json:"rules"`
}

type FlowAction struct {
	Status uint   `json:"status"`
	Body   string `json:"body"`
}

type FlowRule struct {
	KeyType string         `json:"limit_key_type"`
	Action  FlowAction     `json:"action"`
	Rule    *flow.FlowRule `json:"rule"`
}

func init() {
	api.RegisterStream(alipayLimitFilterName, createRpcFlowControlFilterFactory)
}

// StreamFilterFactory represents the stream filter factory.
type StreamFilterFactory struct{}

// CreateFilterChain add the flow control stream filter to filter chain.
func (f *StreamFilterFactory) CreateFilterChain(context context.Context,
	callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewStreamFilter(defaultCallbacks)
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
}

func createRpcFlowControlFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	flowControlCfg := &FlowControlConfig{}
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
	rules := []*flow.FlowRule{}
	for _, rule := range flowControlCfg.Rules {
		rules = append(rules, rule.Rule)
	}
	_, err = flow.LoadRules(rules)
	if err != nil {
		log.DefaultLogger.Errorf("update rules failed")
		return nil, err
	}
	factory := &StreamFilterFactory{}
	return factory, nil
}
