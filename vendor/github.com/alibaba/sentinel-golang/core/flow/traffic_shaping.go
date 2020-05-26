package flow

import (
	"github.com/alibaba/sentinel-golang/core/base"
)

// TrafficShapingCalculator calculates the actual traffic shaping threshold
// based on the threshold of rule and the traffic shaping strategy.
type TrafficShapingCalculator interface {
	CalculateAllowedTokens(node base.StatNode, acquireCount uint32, flag int32) float64
}

// TrafficShapingChecker performs checking according to current metrics and the traffic
// shaping strategy, then yield the token result.
type TrafficShapingChecker interface {
	DoCheck(node base.StatNode, acquireCount uint32, threshold float64) *base.TokenResult
}

type TrafficShapingController struct {
	flowCalculator TrafficShapingCalculator
	flowChecker    TrafficShapingChecker

	rule *FlowRule
}

// NewTrafficShapingController creates a TrafficShapingController wrapped with the given checker and flow rule.
func NewTrafficShapingController(flowCalculator TrafficShapingCalculator, flowChecker TrafficShapingChecker, rule *FlowRule) *TrafficShapingController {
	return &TrafficShapingController{flowCalculator: flowCalculator, flowChecker: flowChecker, rule: rule}
}

func (t *TrafficShapingController) Rule() *FlowRule {
	return t.rule
}

func (t *TrafficShapingController) FlowChecker() TrafficShapingChecker {
	return t.flowChecker
}

func (t *TrafficShapingController) FlowCalculator() TrafficShapingCalculator {
	return t.flowCalculator
}

func (t *TrafficShapingController) PerformChecking(node base.StatNode, acquireCount uint32, flag int32) *base.TokenResult {
	allowedTokens := t.flowCalculator.CalculateAllowedTokens(node, acquireCount, flag)
	return t.flowChecker.DoCheck(node, acquireCount, allowedTokens)
}
