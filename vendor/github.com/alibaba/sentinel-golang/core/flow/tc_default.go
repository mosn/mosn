package flow

import (
	"github.com/alibaba/sentinel-golang/core/base"
)

type DefaultTrafficShapingCalculator struct {
	threshold float64
}

func NewDefaultTrafficShapingCalculator(threshold float64) *DefaultTrafficShapingCalculator {
	return &DefaultTrafficShapingCalculator{threshold: threshold}
}

func (d *DefaultTrafficShapingCalculator) CalculateAllowedTokens(base.StatNode, uint32, int32) float64 {
	return d.threshold
}

type DefaultTrafficShapingChecker struct {
	metricType MetricType
}

func NewDefaultTrafficShapingChecker(metricType MetricType) *DefaultTrafficShapingChecker {
	return &DefaultTrafficShapingChecker{metricType: metricType}
}

func (d *DefaultTrafficShapingChecker) DoCheck(node base.StatNode, acquireCount uint32, threshold float64) *base.TokenResult {
	if node == nil {
		return base.NewTokenResultPass()
	}
	var curCount float64
	if d.metricType == Concurrency {
		curCount = float64(node.CurrentGoroutineNum())
	} else {
		curCount = node.GetQPS(base.MetricEventPass)
	}
	if curCount+float64(acquireCount) > threshold {
		return base.NewTokenResultBlocked(base.BlockTypeFlow, "Flow")
	}
	return base.NewTokenResultPass()
}
