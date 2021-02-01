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

package api

import (
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/alibaba/sentinel-golang/core/hotspot"
	"github.com/alibaba/sentinel-golang/core/isolation"
	"github.com/alibaba/sentinel-golang/core/log"
	"github.com/alibaba/sentinel-golang/core/misc"
	"github.com/alibaba/sentinel-golang/core/stat"
	"github.com/alibaba/sentinel-golang/core/system"
)

var globalSlotChain = BuildDefaultSlotChain()

func GlobalSlotChain() *base.SlotChain {
	return globalSlotChain
}

func BuildDefaultSlotChain() *base.SlotChain {
	sc := base.NewSlotChain()
	sc.AddStatPrepareSlot(stat.DefaultResourceNodePrepareSlot)

	sc.AddRuleCheckSlot(system.DefaultAdaptiveSlot)
	sc.AddRuleCheckSlot(flow.DefaultSlot)
	sc.AddRuleCheckSlot(isolation.DefaultSlot)
	sc.AddRuleCheckSlot(hotspot.DefaultSlot)
	sc.AddRuleCheckSlot(circuitbreaker.DefaultSlot)

	sc.AddStatSlot(stat.DefaultSlot)
	sc.AddStatSlot(log.DefaultSlot)
	sc.AddStatSlot(flow.DefaultStandaloneStatSlot)
	sc.AddStatSlot(hotspot.DefaultConcurrencyStatSlot)
	sc.AddStatSlot(circuitbreaker.DefaultMetricStatSlot)
	return sc
}

// RegisterGlobalStatPrepareSlot registers the global StatPrepareSlot for all resource
// Note: this function is not thread-safe
func RegisterGlobalStatPrepareSlot(slot base.StatPrepareSlot) {
	misc.RegisterGlobalStatPrepareSlot(slot)
}

// RegisterGlobalRuleCheckSlot registers the global RuleCheckSlot for all resource
// Note: this function is not thread-safe
func RegisterGlobalRuleCheckSlot(slot base.RuleCheckSlot) {
	misc.RegisterGlobalRuleCheckSlot(slot)
}

// RegisterGlobalStatSlot registers the global StatSlot for all resource
// Note: this function is not thread-safe
func RegisterGlobalStatSlot(slot base.StatSlot) {
	misc.RegisterGlobalStatSlot(slot)
}
