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

package metrix

import (
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/limit"
	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
)

func TestNewStat(t *testing.T) {
	limitConfig := model.LimitConfig{
		LimitStrategy: limit.QPSStrategy,
		MaxBurstRatio: 1.0,
		PeriodMs:      1000,
		MaxAllows:     10,
	}
	ruleConfig := &model.RuleConfig{
		LimitConfig: limitConfig,
		Id:          99,
		Name:        "test_rule",
		RunMode:     model.RunModeControl,
	}
	stat := NewStat(ruleConfig)
	countPlus(stat)

	time.Sleep(5 * time.Second)

	countPlus(stat)
}

func countPlus(stat *Stat) {
	ticker := NewTicker(func() {
		stat.Counter(INVOKE).Inc(2)
		stat.Counter(BLOCK).Inc(1)
	})
	ticker.Start(time.Millisecond * 100)
	time.Sleep(5 * time.Second)
	ticker.Stop()
}
