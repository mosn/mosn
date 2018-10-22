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

	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/limit"
	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/metrix"
	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/resource"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
)

var params = []model.ComparisonCofig{
	{
		Key:         "aa",
		Value:       "va",
		CompareType: resource.CompareEquals,
	},
	{
		Key:         "bb",
		Value:       "vb",
		CompareType: resource.CompareEquals,
	},
}

var resourceConfig = model.ResourceConfig{
	Headers: []model.ComparisonCofig{
		{
			CompareType: resource.CompareEquals,
			Key:         protocol.MosnHeaderPathKey,
			Value:       "/serverlist/xx.do",
		},
	},
	Params: params,
}

var limitConfig = model.LimitConfig{
	LimitStrategy: limit.QPSStrategy,
	MaxBurstRatio: 1.0,
	PeriodMs:      1000,
	MaxAllows:     10,
}

var ruleConfig = model.RuleConfig{
	Id:              9,
	Name:            "test_666",
	Enable:          true,
	RunMode:         model.RunModeControl,
	ResourceConfigs: []model.ResourceConfig{resourceConfig},
	LimitConfig:     limitConfig,
}

var headers = protocol.CommonHeader{
	protocol.MosnHeaderPathKey:        "/serverlist/xx.do",
	protocol.MosnHeaderQueryStringKey: "aa=va&&bb=vb",
}

func TestNewRuleEngine(t *testing.T) {
	ruleEngine := NewRuleEngine(&ruleConfig)

	{
		log.DefaultLogger.Infof("start ticker")
		ticker := metrix.NewTicker(func() {
			ruleEngine.invoke(headers)
		})
		ticker.Start(time.Millisecond * 10)
		time.Sleep(5 * time.Second)
		ticker.Stop()
		log.DefaultLogger.Infof("stop ticker")
	}

	ruleEngine.stop()
	log.DefaultLogger.Infof("stop ruleEngine")
	time.Sleep(2 * time.Second)
	log.DefaultLogger.Infof("end")
}
