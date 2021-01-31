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

package circuitbreaker

import (
	"github.com/alibaba/sentinel-golang/core/base"
)

const (
	RuleCheckSlotName  = "sentinel-core-circuit-breaker-rule-check-slot"
	RuleCheckSlotOrder = 5000
)

var (
	DefaultSlot = &Slot{}
)

type Slot struct {
}

func (s *Slot) Name() string {
	return RuleCheckSlotName
}

func (s *Slot) Order() uint32 {
	return RuleCheckSlotOrder
}

func (b *Slot) Check(ctx *base.EntryContext) *base.TokenResult {
	resource := ctx.Resource.Name()
	result := ctx.RuleCheckResult
	if len(resource) == 0 {
		return result
	}
	if passed, rule := checkPass(ctx); !passed {
		msg := "circuit breaker check blocked"
		if result == nil {
			result = base.NewTokenResultBlockedWithCause(base.BlockTypeCircuitBreaking, msg, rule, nil)
		} else {
			result.ResetToBlockedWithCause(base.BlockTypeCircuitBreaking, msg, rule, nil)
		}
	}
	return result
}

func checkPass(ctx *base.EntryContext) (bool, *Rule) {
	breakers := getBreakersOfResource(ctx.Resource.Name())
	for _, breaker := range breakers {
		passed := breaker.TryPass(ctx)
		if !passed {
			return false, breaker.BoundRule()
		}
	}
	return true, nil
}
