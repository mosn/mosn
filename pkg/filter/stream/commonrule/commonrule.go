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
	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/types"
	"context"
	"encoding/json"
	"github.com/alipay/sofa-mosn/pkg/log"
	"strconv"
    "github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
)

func init()  {
	filter.RegisterStream("commonrule", CreateCommonRuleFilterFactory)
}

func ParseCommonRuleConfig(config map[string]interface{}) *model.CommonRuleConfig {
	commonRuleConfig := &model.CommonRuleConfig{}

	if data, err := json.Marshal(config); err == nil {
		json.Unmarshal(data, commonRuleConfig)
	} else {
		log.StartLogger.Fatalln("[commonrule] parsing commonRule filter check failed")
	}
	return commonRuleConfig;
}

type commmonRuleFilter struct {
	context        context.Context
	cb types.StreamReceiverFilterCallbacks
	commonRuleConfig *model.CommonRuleConfig
	RuleEngineFactory *RuleEngineFactory
}

var factoryInstance *RuleEngineFactory

func NewFacatoryInstance(config *model.CommonRuleConfig) {
	factoryInstance = NewRuleEngineFactory(config)
	log.DefaultLogger.Infof("newFacatoryInstance:", factoryInstance)
}

func NewCommonRuleFilter(context context.Context, config *model.CommonRuleConfig) types.StreamReceiverFilter {
	f := &commmonRuleFilter{
		context:context,
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
	} else {
		headers.Set(types.HeaderStatus, strconv.Itoa(types.LimitExceededCode))
		f.cb.AppendHeaders(headers, true)
		return types.StreamHeadersFilterStop
	}
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
//end implement StreamReceiverFilter

type CommonRuleFilterFactory struct {
	CommonRuleConfig *model.CommonRuleConfig
}

func (f *CommonRuleFilterFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := NewCommonRuleFilter(context, f.CommonRuleConfig)
	callbacks.AddStreamReceiverFilter(filter)
}

func CreateCommonRuleFilterFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	f := &CommonRuleFilterFactory{
		CommonRuleConfig: ParseCommonRuleConfig(conf),
	}
	NewFacatoryInstance(f.CommonRuleConfig)
	return f, nil
}