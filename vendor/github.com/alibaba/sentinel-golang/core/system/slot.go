package system

import (
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/stat"
)

const SlotName = "SystemAdaptiveSlot"

type SystemAdaptiveSlot struct {
}

func (s *SystemAdaptiveSlot) Check(ctx *base.EntryContext) *base.TokenResult {
	if ctx == nil || ctx.Resource == nil || ctx.Resource.FlowType() != base.Inbound {
		return base.NewTokenResultPass()
	}
	rules := GetRules()
	for _, rule := range rules {
		passed, m := s.doCheckRule(rule)
		if passed {
			continue
		}
		return base.NewTokenResultBlockedWithCause(base.BlockTypeSystemFlow, base.BlockTypeSystemFlow.String(), rule, m)
	}
	return base.NewTokenResultPass()
}

func (s *SystemAdaptiveSlot) doCheckRule(rule *SystemRule) (bool, float64) {
	threshold := rule.TriggerCount
	switch rule.MetricType {
	case InboundQPS:
		qps := stat.InboundNode().GetQPS(base.MetricEventPass)
		res := qps < threshold
		return res, qps
	case Concurrency:
		n := float64(stat.InboundNode().CurrentGoroutineNum())
		res := n < threshold
		return res, n
	case AvgRT:
		rt := stat.InboundNode().AvgRT()
		res := rt < threshold
		return res, rt
	case Load:
		l := CurrentLoad()
		if l > threshold {
			if rule.Strategy != BBR || !checkBbrSimple() {
				return false, l
			}
		}
		return true, l
	case CpuUsage:
		c := CurrentCpuUsage()
		if c > threshold {
			if rule.Strategy != BBR || !checkBbrSimple() {
				return false, c
			}
		}
		return true, c
	default:
		return true, 0
	}
}

func checkBbrSimple() bool {
	concurrency := stat.InboundNode().CurrentGoroutineNum()
	minRt := stat.InboundNode().MinRT()
	maxComplete := stat.InboundNode().GetMaxAvg(base.MetricEventComplete)
	if concurrency > 1 && float64(concurrency) > maxComplete*minRt/1000 {
		return false
	}
	return true
}

func (s *SystemAdaptiveSlot) String() string {
	return SlotName
}
