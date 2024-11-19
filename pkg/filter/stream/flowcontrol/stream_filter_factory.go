/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package flowcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/flow"
	"mosn.io/api"
	datasource "mosn.io/mosn/pkg/filter/stream/flowcontrol/data_source"
	"mosn.io/mosn/pkg/log"
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
	AppName              string                            `json:"app_name"`
	CallbackName         string                            `json:"callback_name"`
	LogPath              string                            `json:"log_path"`
	GlobalSwitch         bool                              `json:"global_switch"`
	Monitor              bool                              `json:"monitor"`
	KeyType              api.ProtocolResourceName          `json:"limit_key_type"`
	Action               Action                            `json:"action"`
	Rules                []*flow.Rule                      `json:"rules"`
	DynamicDataResources []*datasource.DynamicDataResource `json:"dynamic_data_resources"`
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
	config             *Config
	trafficType        base.TrafficType
	dataSourceFactorys map[datasource.ResourceType]datasource.DataSourceFactory
}

// CreateFilterChain add the flow control stream filter to filter chain.
func (f *StreamFilterFactory) CreateFilterChain(context context.Context,
	callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewStreamFilter(GetCallbacksByConfig(f.config), f.trafficType)
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

func defaultConfig() *Config {
	return &Config{
		AppName: defaultSentinelAppName,
		LogPath: defaultSentinelLogDir,
		Action: Action{
			Status: api.LimitExceededCode,
			Body:   defaultResponse,
		},
		KeyType: api.PATH,
	}
}

func loadConfig(conf map[string]interface{}) (*Config, error) {
	flowControlCfg := defaultConfig()
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
	return flowControlCfg, nil
}

func (f *StreamFilterFactory) initFlowRules(flowControlCfg *Config) error {
	if len(flowControlCfg.Rules) > 0 {
		_, err := flow.LoadRules(flowControlCfg.Rules)
		if err != nil {
			return err
		}
	}

	for _, dynamicResources := range flowControlCfg.DynamicDataResources {
		dataSourceFactory, ok := f.dataSourceFactorys[datasource.ResourceType(dynamicResources.ResourceType)]
		if !ok {
			dc, ok := datasource.GetDataSourceFactoryCreator(datasource.ResourceType(dynamicResources.ResourceType))
			if !ok {
				log.DefaultLogger.Warnf("flowcontrol %s  data source doesn't support now", dynamicResources.ResourceType)
				continue
			}

			var err error
			dataSourceFactory, err = dc(flowControlCfg.AppName)
			if err != nil {
				log.DefaultLogger.Errorf("flowcontrol create %s data source factory error: %v", dynamicResources.ResourceType, err)
				return err
			}

			f.dataSourceFactorys[datasource.ResourceType(dynamicResources.ResourceType)] = dataSourceFactory
		}

		dataSource, err := dataSourceFactory.CreateDataSource(dynamicResources.Config)
		if err != nil {
			log.DefaultLogger.Errorf("flowcontrol create %s data source error: %v", dynamicResources.ResourceType, err)
			return err
		}

		err = dataSource.InitFlowRules()
		if err != nil {
			log.DefaultLogger.Errorf("flowcontrol %s data source init flowrules error: %v", dynamicResources.ResourceType, err)
			return err
		}
	}

	return nil
}

func createRpcFlowControlFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	flowControlCfg, err := loadConfig(conf)
	if err != nil {
		return nil, err
	}

	factory := &StreamFilterFactory{
		config:             flowControlCfg,
		trafficType:        parseTrafficType(conf),
		dataSourceFactorys: make(map[datasource.ResourceType]datasource.DataSourceFactory),
	}

	err = factory.initFlowRules(flowControlCfg)
	if err != nil {
		log.DefaultLogger.Errorf("init flow rules failed")
		return nil, err
	}
	initOnce.Do(func() {
		// TODO: can't support dynamically update at present, should be optimized
		initSentinel(flowControlCfg.AppName, flowControlCfg.LogPath)
	})

	return factory, nil
}

func parseTrafficType(conf map[string]interface{}) base.TrafficType {
	directionConf, ok := conf["direction"].(string)
	if !ok {
		return base.Inbound
	}
	if strings.EqualFold(directionConf, base.Outbound.String()) {
		return base.Outbound
	}
	return base.Inbound
}

func isValidConfig(cfg *Config) (bool, error) {
	switch cfg.KeyType {
	case api.PATH, api.ARG, api.URI:
	default:
		return false, errors.New("invalid key type")
	}
	return true, nil
}
