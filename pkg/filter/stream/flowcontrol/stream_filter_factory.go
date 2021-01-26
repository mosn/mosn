package flowcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/flow"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

const defaultResponse = "current request is limited"

// the environment variables.
const (
	// TODO: update log sentinel configurations more graceful.
	envKeySentinelLogDir   = "SENTINEL_LOG_DIR"
	envKeySentinelAppName  = "SENTINEL_APP_NAME"
	defaultSentinelLogDir  = "/tmp/sentinel"
	defaultSentinelAppName = "unknown"
)

var (
	initOnce sync.Once
)

// Config represents the flow control configurations.
type Config struct {
	AppName      string                   `json:"app_name"`
	LogPath      string                   `json:"log_path"`
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

func initSentinel(appName, logPath string) {
	os.Setenv(envKeySentinelAppName, appName)
	os.Setenv(envKeySentinelLogDir, logPath)
	err := sentinel.InitDefault()
	if err != nil {
		log.StartLogger.Errorf("init sentinel failed, error: %v", err)
	}
}

func createRpcFlowControlFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	flowControlCfg := &Config{
		AppName: defaultSentinelAppName,
		LogPath: defaultSentinelLogDir,
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
	err = json.Unmarshal(cfg, flowControlCfg)
	if err != nil {
		log.DefaultLogger.Errorf("parse flow control filter config failed")
		return nil, err
	}
	_, err = isValidConfig(flowControlCfg)
	if err != nil {
		log.DefaultLogger.Errorf("invalid configuration: %v", err)
		return nil, err
	}
	_, err = flow.LoadRules(flowControlCfg.Rules)
	if err != nil {
		log.DefaultLogger.Errorf("update rules failed")
		return nil, err
	}
	initOnce.Do(func() {
		// TODO: can't support dynamically update at present, should be optimized
		initSentinel(flowControlCfg.AppName, flowControlCfg.LogPath)
	})
	factory := &StreamFilterFactory{config: flowControlCfg}
	return factory, nil
}

func isValidConfig(cfg *Config) (bool, error) {
	switch cfg.KeyType {
	case api.PATH, api.ARG, api.URI:
	default:
		return false, errors.New("invalid key type")
	}
	return true, nil
}
