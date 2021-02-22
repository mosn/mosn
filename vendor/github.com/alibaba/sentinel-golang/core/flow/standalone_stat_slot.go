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
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/pkg/errors"
)

const (
	StatSlotName  = "sentinel-core-flow-standalone-stat-slot"
	StatSlotOrder = 3000
)

var (
	DefaultStandaloneStatSlot = &StandaloneStatSlot{}
)

type StandaloneStatSlot struct {
}

func (s *StandaloneStatSlot) Name() string {
	return StatSlotName
}

func (s *StandaloneStatSlot) Order() uint32 {
	return StatSlotOrder
}

func (s StandaloneStatSlot) OnEntryPassed(ctx *base.EntryContext) {
	res := ctx.Resource.Name()
	for _, tc := range getTrafficControllerListFor(res) {
		if !tc.boundStat.reuseResourceStat {
			if tc.boundStat.writeOnlyMetric != nil {
				tc.boundStat.writeOnlyMetric.AddCount(base.MetricEventPass, int64(ctx.Input.BatchCount))
			} else {
				logging.Error(errors.New("nil independent write statistic"), "Nil statistic for traffic control in StandaloneStatSlot.OnEntryPassed()", "rule", tc.rule)
			}
		}
	}
}

func (s StandaloneStatSlot) OnEntryBlocked(ctx *base.EntryContext, blockError *base.BlockError) {
	// Do nothing
}

func (s StandaloneStatSlot) OnCompleted(ctx *base.EntryContext) {
	// Do nothing
}
