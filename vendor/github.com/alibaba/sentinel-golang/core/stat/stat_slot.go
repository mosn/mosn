package stat

import (
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/util"
)

const SlotName = "StatisticSlot"

type StatisticSlot struct {
}

func (s *StatisticSlot) String() string {
	return SlotName
}

func (s *StatisticSlot) OnEntryPassed(ctx *base.EntryContext) {
	s.recordPassFor(ctx.StatNode, ctx.Input.AcquireCount)
	if ctx.Resource.FlowType() == base.Inbound {
		s.recordPassFor(InboundNode(), ctx.Input.AcquireCount)
	}
}

func (s *StatisticSlot) OnEntryBlocked(ctx *base.EntryContext, blockError *base.BlockError) {
	s.recordBlockFor(ctx.StatNode, ctx.Input.AcquireCount)
	if ctx.Resource.FlowType() == base.Inbound {
		s.recordBlockFor(InboundNode(), ctx.Input.AcquireCount)
	}
}

func (s *StatisticSlot) OnCompleted(ctx *base.EntryContext) {
	if ctx.Output.LastResult == nil || ctx.Output.LastResult.IsBlocked() {
		return
	}
	rt := util.CurrentTimeMillis() - ctx.StartTime()
	s.recordCompleteFor(ctx.StatNode, ctx.Input.AcquireCount, rt)
	if ctx.Resource.FlowType() == base.Inbound {
		s.recordCompleteFor(InboundNode(), ctx.Input.AcquireCount, rt)
	}
}

func (s *StatisticSlot) recordPassFor(sn base.StatNode, count uint32) {
	if sn == nil {
		return
	}
	sn.IncreaseGoroutineNum()
	sn.AddMetric(base.MetricEventPass, uint64(count))
}

func (s *StatisticSlot) recordBlockFor(sn base.StatNode, count uint32) {
	if sn == nil {
		return
	}
	sn.AddMetric(base.MetricEventBlock, uint64(count))
}

func (s *StatisticSlot) recordCompleteFor(sn base.StatNode, count uint32, rt uint64) {
	if sn == nil {
		return
	}
	sn.AddMetric(base.MetricEventRt, rt)
	sn.AddMetric(base.MetricEventComplete, uint64(count))
	sn.DecreaseGoroutineNum()
}
