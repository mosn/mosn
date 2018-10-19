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
	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// RuleEngineFactory as
type RuleEngineFactory struct {
	CommonRuleConfig *model.CommonRuleConfig
	ruleEngines      []RuleEngine
}

// NewRuleEngineFactory new
func NewRuleEngineFactory(config *model.CommonRuleConfig) *RuleEngineFactory {
	f := &RuleEngineFactory{
		CommonRuleConfig: config,
	}

	for _, ruleConfig := range config.RuleConfigs {
		if ruleConfig.Enable {
			ruleEngine := NewRuleEngine(&ruleConfig)
			if ruleEngine != nil {
				f.ruleEngines = append(f.ruleEngines, *ruleEngine)
			}
		}
	}

	return f
}

func (f *RuleEngineFactory) invoke(headers types.HeaderMap) bool {
	for _, ruleEngine := range f.ruleEngines {
		if !ruleEngine.invoke(headers) {
			return false
		}
	}
	return true
}

func (f *RuleEngineFactory) stop() {
	for _, ruleEngine := range f.ruleEngines {
		ruleEngine.stop()
	}
}
