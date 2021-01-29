// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package flow

import (
	"github.com/alibaba/sentinel-golang/core/base"
)

type DirectTrafficShapingCalculator struct {
	owner     *TrafficShapingController
	threshold float64
}

func NewDirectTrafficShapingCalculator(owner *TrafficShapingController, threshold float64) *DirectTrafficShapingCalculator {
	return &DirectTrafficShapingCalculator{
		owner:     owner,
		threshold: threshold,
	}
}

func (d *DirectTrafficShapingCalculator) CalculateAllowedTokens(uint32, int32) float64 {
	return d.threshold
}

func (d *DirectTrafficShapingCalculator) BoundOwner() *TrafficShapingController {
	return d.owner
}

type RejectTrafficShapingChecker struct {
	owner *TrafficShapingController
	rule  *Rule
}

func NewRejectTrafficShapingChecker(owner *TrafficShapingController, rule *Rule) *RejectTrafficShapingChecker {
	return &RejectTrafficShapingChecker{
		owner: owner,
		rule:  rule,
	}
}

func (d *RejectTrafficShapingChecker) BoundOwner() *TrafficShapingController {
	return d.owner
}

func (d *RejectTrafficShapingChecker) DoCheck(resStat base.StatNode, batchCount uint32, threshold float64) *base.TokenResult {
	metricReadonlyStat := d.BoundOwner().boundStat.readOnlyMetric
	if metricReadonlyStat == nil {
		return nil
	}
	curCount := float64(metricReadonlyStat.GetSum(base.MetricEventPass))
	if curCount+float64(batchCount) > threshold {
		msg := "flow reject check blocked"
		return base.NewTokenResultBlockedWithCause(base.BlockTypeFlow, msg, d.rule, curCount)
	}
	return nil
}
