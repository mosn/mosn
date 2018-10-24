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

package limit

import (
	"errors"

	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
	"github.com/alipay/sofa-mosn/pkg/log"
)

// LimitEngine limit
type LimitEngine struct {
	RuleConfig *model.RuleConfig
	limiter    Limiter
}

// NewLimitEngine limit
func NewLimitEngine(ruleConfig *model.RuleConfig) (*LimitEngine, error) {
	l := &LimitEngine{
		RuleConfig: ruleConfig,
	}
	config := ruleConfig.LimitConfig
	if config.LimitStrategy == QPSStrategy {
		limiter, err := NewQPSLimiter(int64(config.MaxAllows), int64(config.PeriodMs))
		if err != nil {
			log.DefaultLogger.Errorf("create NewQPSLimiter error, err: %s", err)
			return nil, err
		}
		l.limiter = limiter
		return l, nil
	} else if config.LimitStrategy == RateLimiterStrategy {
		limiter, err := NewRateLimiter(int64(config.MaxAllows), int64(config.PeriodMs), float64(config.MaxBurstRatio))
		if err != nil {
			log.DefaultLogger.Errorf("create NewRateLimiter error, err: %s", err)
			return nil, err
		}
		l.limiter = limiter
		return l, nil
	}
	return nil, errors.New("Unknown LimitStrategy type:" + config.LimitStrategy)
}

// OverLimit check limit
func (engine *LimitEngine) OverLimit() bool {
	return !engine.limiter.TryAcquire()
}
