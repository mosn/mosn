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

package stat

import (
	"sync/atomic"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/config"
	sbase "github.com/alibaba/sentinel-golang/core/stat/base"
)

type BaseStatNode struct {
	sampleCount uint32
	intervalMs  uint32

	concurrency int32

	arr    *sbase.BucketLeapArray
	metric *sbase.SlidingWindowMetric
}

func NewBaseStatNode(sampleCount uint32, intervalInMs uint32) *BaseStatNode {
	la := sbase.NewBucketLeapArray(config.GlobalStatisticSampleCountTotal(), config.GlobalStatisticIntervalMsTotal())
	metric, _ := sbase.NewSlidingWindowMetric(sampleCount, intervalInMs, la)
	return &BaseStatNode{
		concurrency: 0,
		sampleCount: sampleCount,
		intervalMs:  intervalInMs,
		arr:         la,
		metric:      metric,
	}
}

// NewCustomizedBaseStatNode creates the BaseStatNode with specified sampleCount
// and intervalInMs.
func NewCustomizedBaseStatNode(sampleCount uint32, intervalInMs uint32) (*BaseStatNode, error) {
	la := sbase.NewBucketLeapArray(sampleCount, intervalInMs)
	metric, err := sbase.NewSlidingWindowMetric(sampleCount, intervalInMs, la)
	if err != nil {
		return nil, err
	}
	return &BaseStatNode{
		concurrency: 0,
		sampleCount: sampleCount,
		intervalMs:  intervalInMs,
		arr:         la,
		metric:      metric,
	}, nil
}

func (n *BaseStatNode) MetricsOnCondition(predicate base.TimePredicate) []*base.MetricItem {
	return n.metric.SecondMetricsOnCondition(predicate)
}

func (n *BaseStatNode) GetQPS(event base.MetricEvent) float64 {
	return n.metric.GetQPS(event)
}

func (n *BaseStatNode) GetPreviousQPS(event base.MetricEvent) float64 {
	return n.metric.GetPreviousQPS(event)
}

func (n *BaseStatNode) GetSum(event base.MetricEvent) int64 {
	return n.metric.GetSum(event)
}

func (n *BaseStatNode) GetMaxAvg(event base.MetricEvent) float64 {
	return float64(n.metric.GetMaxOfSingleBucket(event)) * float64(n.sampleCount) / float64(n.intervalMs) * 1000.0
}

func (n *BaseStatNode) AddCount(event base.MetricEvent, count int64) {
	n.arr.AddCount(event, count)
}

func (n *BaseStatNode) AvgRT() float64 {
	complete := n.metric.GetSum(base.MetricEventComplete)
	if complete <= 0 {
		return float64(0.0)
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

func (n *BaseStatNode) CurrentConcurrency() int32 {
	return atomic.LoadInt32(&(n.concurrency))
}

func (n *BaseStatNode) IncreaseConcurrency() {
	atomic.AddInt32(&(n.concurrency), 1)
	n.arr.UpdateMaxConcurrency(int64(n.CurrentConcurrency()))
}

func (n *BaseStatNode) DecreaseConcurrency() {
	atomic.AddInt32(&(n.concurrency), -1)
}

func (n *BaseStatNode) GenerateReadStat(sampleCount uint32, intervalInMs uint32) (base.ReadStat, error) {
	return sbase.NewSlidingWindowMetric(sampleCount, intervalInMs, n.arr)
}

func (n *BaseStatNode) DefaultMetric() base.ReadStat {
	return n.metric
}
