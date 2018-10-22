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
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/metrix"
	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
	"github.com/alipay/sofa-mosn/pkg/log"
)

func TestNewRuleEngineFactory(t *testing.T) {
	commonRuleConfig := &model.CommonRuleConfig{
		RuleConfigs: []model.RuleConfig{ruleConfig},
	}
	ruleEngineFactory := NewRuleEngineFactory(commonRuleConfig)

	total := 0
	success := 0
	fail := 0

	{
		log.DefaultLogger.Infof("start ticker")
		ticker := metrix.NewTicker(func() {
			total++
			if ruleEngineFactory.invoke(headers) {
				success++
			} else {
				fail++
			}
		})
		ticker.Start(time.Millisecond * 10)
		time.Sleep(5 * time.Second)
		ticker.Stop()
		log.DefaultLogger.Infof("stop ticker")
	}

	ruleEngineFactory.stop()
	log.DefaultLogger.Infof("stop ruleEngineFactory")
	log.DefaultLogger.Infof("total=%d, success=%d, fail=%d", total, success, fail)
	time.Sleep(2 * time.Second)
	log.DefaultLogger.Infof("end")
}
