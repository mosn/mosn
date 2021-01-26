package api

import (
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/alibaba/sentinel-golang/core/log"
	"github.com/alibaba/sentinel-golang/core/stat"
	"github.com/alibaba/sentinel-golang/core/system"
)

var globalSlotChain = BuildDefaultSlotChain()

// SetSlotChain replaces current slot chain with the given one.
// Note that this operation is not thread-safe, so it should be
// called when pre-initializing Sentinel.
func SetSlotChain(chain *base.SlotChain) {
	if chain != nil {
		globalSlotChain = chain
	}
}

func GlobalSlotChain() *base.SlotChain {
	return globalSlotChain
}

func BuildDefaultSlotChain() *base.SlotChain {
	sc := base.NewSlotChain()
	sc.AddStatPrepareSlotLast(&stat.StatNodePrepareSlot{})
	sc.AddRuleCheckSlotLast(&system.SystemAdaptiveSlot{})
	sc.AddRuleCheckSlotLast(&flow.FlowSlot{})
	sc.AddStatSlotLast(&stat.StatisticSlot{})
	sc.AddStatSlotLast(&log.LogSlot{})
	return sc
}
