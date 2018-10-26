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

package commonrule

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	filter.RegisterStream("commonrule", CreateCommonRuleFilterFactory)
}

func parseCommonRuleConfig(config map[string]interface{}) *model.CommonRuleConfig {
	commonRuleConfig := &model.CommonRuleConfig{}

	if data, err := json.Marshal(config); err == nil {
		json.Unmarshal(data, commonRuleConfig)
	} else {
		log.StartLogger.Fatalln("[commonrule] parsing commonRule filter check failed")
	}
	return commonRuleConfig
}

type commmonRuleFilter struct {
	context           context.Context
	cb                types.StreamReceiverFilterCallbacks
	commonRuleConfig  *model.CommonRuleConfig
	RuleEngineFactory *RuleEngineFactory
}

var factoryInstance *RuleEngineFactory

// NewFacatoryInstance as
func NewFacatoryInstance(config *model.CommonRuleConfig) {
	factoryInstance = NewRuleEngineFactory(config)
	log.DefaultLogger.Infof("newFacatoryInstance:", factoryInstance)
}

// NewCommonRuleFilter as
func NewCommonRuleFilter(context context.Context, config *model.CommonRuleConfig) types.StreamReceiverFilter {
	f := &commmonRuleFilter{
		context:          context,
		commonRuleConfig: config,
	}
	f.RuleEngineFactory = factoryInstance
	return f
}

//implement StreamReceiverFilter
func (f *commmonRuleFilter) OnDecodeHeaders(headers types.HeaderMap, endStream bool) types.StreamHeadersFilterStatus {
	// do filter
	if f.RuleEngineFactory.invoke(headers) {
		return types.StreamHeadersFilterContinue
	}
	headers.Set(types.HeaderStatus, strconv.Itoa(types.LimitExceededCode))
	f.cb.AppendHeaders(headers, true)
	return types.StreamHeadersFilterStop
}

func (f *commmonRuleFilter) OnDecodeData(buf types.IoBuffer, endStream bool) types.StreamDataFilterStatus {
	//do filter
	return types.StreamDataFilterContinue
}

func (f *commmonRuleFilter) OnDecodeTrailers(trailers types.HeaderMap) types.StreamTrailersFilterStatus {
	//do filter
	return types.StreamTrailersFilterContinue
}

func (f *commmonRuleFilter) SetDecoderFilterCallbacks(cb types.StreamReceiverFilterCallbacks) {
	f.cb = cb
}

func (f *commmonRuleFilter) OnDestroy() {}

type commonRuleFilterFactory struct {
	commonRuleConfig *model.CommonRuleConfig
}

func (f *commonRuleFilterFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := NewCommonRuleFilter(context, f.commonRuleConfig)
	callbacks.AddStreamReceiverFilter(filter)
}

// CreateCommonRuleFilterFactory as
func CreateCommonRuleFilterFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	f := &commonRuleFilterFactory{
		commonRuleConfig: parseCommonRuleConfig(conf),
	}
	NewFacatoryInstance(f.commonRuleConfig)
	return f, nil
}
