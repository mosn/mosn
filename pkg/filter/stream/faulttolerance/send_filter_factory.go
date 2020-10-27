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

package faulttolerance

import (
	"context"
	"encoding/json"
	"errors"

	"mosn.io/mosn/pkg/filter/stream/faulttolerance/regulator"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/config"
)

func init() {
	api.RegisterStream(v2.FaultTolerance, CreateSendFilterFactory)
}

type SendFilterFactory struct {
	config            *v2.FaultToleranceFilterConfig
	invocationFactory *regulator.InvocationStatFactory
}

func (f *SendFilterFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewSendFilter(f.config, f.invocationFactory)
	callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}

func CreateSendFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	if filterConfig, err := parseConfig(conf); err != nil {
		return nil, err
	} else {
		invocationFactory := regulator.NewInvocationStatFactory(filterConfig)
		return &SendFilterFactory{
			config:            filterConfig,
			invocationFactory: invocationFactory,
		}, nil
	}
}

func parseConfig(cfg map[string]interface{}) (*v2.FaultToleranceFilterConfig, error) {
	ruleJson := &config.FaultToleranceRuleJson{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, ruleJson); err != nil {
		return nil, err
	}

	fillDefaultValue(ruleJson)

	if !isLegal(ruleJson) {
		return nil, errors.New("config is illegal")
	}

	exceptionTypes := map[uint32]bool{}
	for _, exceptionType := range ruleJson.ExceptionTypes {
		exceptionTypes[exceptionType] = true
	}
	filterConfig := &v2.FaultToleranceFilterConfig{
		Enabled:               ruleJson.Enabled,
		ExceptionTypes:        exceptionTypes,
		TimeWindow:            ruleJson.TimeWindow,
		LeastWindowCount:      ruleJson.LeastWindowCount,
		ExceptionRateMultiple: ruleJson.ExceptionRateMultiple,
		MaxIpCount:            ruleJson.MaxIpCount,
		MaxIpRatio:            ruleJson.MaxIpRatio,
		RecoverTime:           ruleJson.RecoverTime,
		TaskSize:              ruleJson.TaskSize,
	}
	return filterConfig, nil
}

func isLegal(ruleJson *config.FaultToleranceRuleJson) bool {
	if ruleJson.TimeWindow < 10 {
		return false
	}
	if ruleJson.LeastWindowCount <= 0 {
		return false
	}
	if ruleJson.MaxIpCount < 0 {
		return false
	}
	if ruleJson.ExceptionRateMultiple <= 1 {
		return false
	}
	return true
}

func fillDefaultValue(ruleJson *config.FaultToleranceRuleJson) {
	if ruleJson.TimeWindow == 0 {
		ruleJson.TimeWindow = 10000
	}
	if ruleJson.ExceptionRateMultiple == 0 {
		ruleJson.ExceptionRateMultiple = 5
	}
	if ruleJson.LeastWindowCount == 0 {
		ruleJson.LeastWindowCount = 10
	}
	if ruleJson.ExceptionTypes == nil {
		ruleJson.ExceptionTypes = []uint32{502, 503, 504}
	}
	if ruleJson.MaxIpCount == 0 {
		ruleJson.MaxIpCount = 1
	}
	if ruleJson.RecoverTime == 0 {
		ruleJson.RecoverTime = 15 * 60000
	}
	if ruleJson.TaskSize == 0 {
		ruleJson.TaskSize = 20
	}
}
