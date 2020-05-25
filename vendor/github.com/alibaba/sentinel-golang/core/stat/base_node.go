package stat

import (
	"sync/atomic"

	"github.com/alibaba/sentinel-golang/core/base"
	sbase "github.com/alibaba/sentinel-golang/core/stat/base"
)

type BaseStatNode struct {
	sampleCount uint32
	intervalMs  uint32

	goroutineNum int32

	arr    *sbase.BucketLeapArray
	metric *sbase.SlidingWindowMetric
}

func NewBaseStatNode(sampleCount uint32, intervalInMs uint32) *BaseStatNode {
	la := sbase.NewBucketLeapArray(base.DefaultSampleCountTotal, base.DefaultIntervalMsTotal)
	metric := sbase.NewSlidingWindowMetric(sampleCount, intervalInMs, la)
	return &BaseStatNode{
		goroutineNum: 0,
		sampleCount:  sampleCount,
		intervalMs:   intervalInMs,
		arr:          la,
		metric:       metric,
	}
}

func (n *BaseStatNode) MetricsOnCondition(predicate base.TimePredicate) []*base.MetricItem {
	return n.metric.SecondMetricsOnCondition(predicate)
}

func (n *BaseStatNode) GetQPS(event base.MetricEvent) float64 {
	return n.metric.GetQPS(event)
}

func (n *BaseStatNode) GetSum(event base.MetricEvent) int64 {
	return n.metric.GetSum(event)
}

func (n *BaseStatNode) GetMaxAvg(event base.MetricEvent) float64 {
	return float64(n.metric.GetMaxOfSingleBucket(event)) * float64(n.sampleCount) / float64(n.intervalMs) * 1000
}

func (n *BaseStatNode) AddMetric(event base.MetricEvent, count uint64) {
	n.arr.AddCount(event, int64(count))
}

func (n *BaseStatNode) AvgRT() float64 {
	complete := n.metric.GetSum(base.MetricEventComplete)
	if complete <= 0 {
		return float64(0)
	}
	return float64(n.metric.GetSum(base.MetricEventRt) / complete)
}

func (n *BaseStatNode) MinRT() float64 {
	return float64(n.metric.MinRT())
}

// MaxConcurrency returns the max concurrency count of whole sliding window.
func (n *BaseStatNode) MaxConcurrency() int64 {
	return n.metric.MaxConcurrency()
}

// SecondMaxConcurrency returns the max concurrency count of latest second.
func (n *BaseStatNode) SecondMaxConcurrency() int64 {
	return n.metric.SecondMaxConcurrency()
}

func (n *BaseStatNode) CurrentGoroutineNum() int32 {
	return atomic.LoadInt32(&(n.goroutineNum))
}

func (n *BaseStatNode) IncreaseGoroutineNum() {
	atomic.AddInt32(&(n.goroutineNum), 1)
	n.arr.UpdateMaxConcurrency(int64(n.CurrentGoroutineNum()))
}

func (n *BaseStatNode) DecreaseGoroutineNum() {
	atomic.AddInt32(&(n.goroutineNum), -1)
}

func (n *BaseStatNode) Reset() {
	// TODO: this should be thread-safe, or error may occur
	panic("to be implemented")
}
