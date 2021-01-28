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

package base

import (
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
)

// SlidingWindowMetric represents the sliding window metric wrapper.
// It does not store any data and is the wrapper of BucketLeapArray to adapt to different internal bucket
// SlidingWindowMetric is used for SentinelRules and BucketLeapArray is used for monitor
// BucketLeapArray is per resource, and SlidingWindowMetric support only read operation.
type SlidingWindowMetric struct {
	bucketLengthInMs uint32
	sampleCount      uint32
	intervalInMs     uint32
	real             *BucketLeapArray
}

// It must pass the parameter point to the real storage entity
func NewSlidingWindowMetric(sampleCount, intervalInMs uint32, real *BucketLeapArray) (*SlidingWindowMetric, error) {
	if real == nil {
		return nil, errors.New("nil BucketLeapArray")
	}
	if err := base.CheckValidityForReuseStatistic(sampleCount, intervalInMs, real.SampleCount(), real.IntervalInMs()); err != nil {
		return nil, err
	}
	bucketLengthInMs := intervalInMs / sampleCount

	return &SlidingWindowMetric{
		bucketLengthInMs: bucketLengthInMs,
		sampleCount:      sampleCount,
		intervalInMs:     intervalInMs,
		real:             real,
	}, nil
}

// Get the start time range of the bucket for the provided time.
// The actual time span is: [start, end + in.bucketTimeLength)
func (m *SlidingWindowMetric) getBucketStartRange(timeMs uint64) (start, end uint64) {
	curBucketStartTime := calculateStartTime(timeMs, m.real.BucketLengthInMs())
	end = curBucketStartTime
	start = end - uint64(m.intervalInMs) + uint64(m.real.BucketLengthInMs())
	return
}

func (m *SlidingWindowMetric) getIntervalInSecond() float64 {
	return float64(m.intervalInMs) / 1000.0
}

func (m *SlidingWindowMetric) count(event base.MetricEvent, values []*BucketWrap) int64 {
	ret := int64(0)
	for _, ww := range values {
		mb := ww.Value.Load()
		if mb == nil {
			logging.Error(errors.New("nil BucketWrap"), "Current bucket value is nil in SlidingWindowMetric.count()")
			continue
		}
		counter, ok := mb.(*MetricBucket)
		if !ok {
			logging.Error(errors.New("type assert failed"), "Fail to do type assert in SlidingWindowMetric.count()", "expectType", "*MetricBucket", "actualType", reflect.TypeOf(mb).Name())
			continue
		}
		ret += counter.Get(event)
	}
	return ret
}

func (m *SlidingWindowMetric) GetSum(event base.MetricEvent) int64 {
	return m.getSumWithTime(util.CurrentTimeMillis(), event)
}

func (m *SlidingWindowMetric) getSumWithTime(now uint64, event base.MetricEvent) int64 {
	satisfiedBuckets := m.getSatisfiedBuckets(now)
	return m.count(event, satisfiedBuckets)
}

func (m *SlidingWindowMetric) GetQPS(event base.MetricEvent) float64 {
	return m.getQPSWithTime(util.CurrentTimeMillis(), event)
}

func (m *SlidingWindowMetric) GetPreviousQPS(event base.MetricEvent) float64 {
	return m.getQPSWithTime(util.CurrentTimeMillis()-uint64(m.bucketLengthInMs), event)
}

func (m *SlidingWindowMetric) getQPSWithTime(now uint64, event base.MetricEvent) float64 {
	return float64(m.getSumWithTime(now, event)) / m.getIntervalInSecond()
}

func (m *SlidingWindowMetric) getSatisfiedBuckets(now uint64) []*BucketWrap {
	start, end := m.getBucketStartRange(now)
	satisfiedBuckets := m.real.ValuesConditional(now, func(ws uint64) bool {
		return ws >= start && ws <= end
	})
	return satisfiedBuckets
}

func (m *SlidingWindowMetric) GetMaxOfSingleBucket(event base.MetricEvent) int64 {
	now := util.CurrentTimeMillis()
	satisfiedBuckets := m.getSatisfiedBuckets(now)
	var curMax int64 = 0
	for _, w := range satisfiedBuckets {
		mb := w.Value.Load()
		if mb == nil {
			logging.Error(errors.New("nil BucketWrap"), "Current bucket value is nil in SlidingWindowMetric.GetMaxOfSingleBucket()")
			continue
		}
		counter, ok := mb.(*MetricBucket)
		if !ok {
			logging.Error(errors.New("type assert failed"), "Fail to do type assert in SlidingWindowMetric.GetMaxOfSingleBucket()", "expectType", "*MetricBucket", "actualType", reflect.TypeOf(mb).Name())
			continue
		}
		v := counter.Get(event)
		if v > curMax {
			curMax = v
		}
	}
	return curMax
}

func (m *SlidingWindowMetric) MinRT() float64 {
	now := util.CurrentTimeMillis()
	satisfiedBuckets := m.getSatisfiedBuckets(now)
	minRt := base.DefaultStatisticMaxRt
	for _, w := range satisfiedBuckets {
		mb := w.Value.Load()
		if mb == nil {
			logging.Error(errors.New("nil BucketWrap"), "Current bucket value is nil in SlidingWindowMetric.MinRT()")
			continue
		}
		counter, ok := mb.(*MetricBucket)
		if !ok {
			logging.Error(errors.New("type assert failed"), "Fail to do type assert in SlidingWindowMetric.MinRT()", "expectType", "*MetricBucket", "actualType", reflect.TypeOf(mb).Name())
			continue
		}
		v := counter.MinRt()
		if v < minRt {
			minRt = v
		}
	}
	if minRt < 1 {
		minRt = 1
	}
	return float64(minRt)
}

func (m *SlidingWindowMetric) AvgRT() float64 {
	return float64(m.GetSum(base.MetricEventRt)) / float64(m.GetSum(base.MetricEventComplete))
}

// SecondMetricsOnCondition aggregates metric items by second on condition that
// the startTime of the statistic buckets satisfies the time predicate.
func (m *SlidingWindowMetric) SecondMetricsOnCondition(predicate base.TimePredicate) []*base.MetricItem {
	ws := m.real.ValuesConditional(util.CurrentTimeMillis(), predicate)

	// Aggregate second-level MetricItem (only for stable metrics)
	wm := make(map[uint64][]*BucketWrap, 8)
	for _, w := range ws {
		bucketStart := atomic.LoadUint64(&w.BucketStart)
		secStart := bucketStart - bucketStart%1000
		if arr, hasData := wm[secStart]; hasData {
			wm[secStart] = append(arr, w)
		} else {
			wm[secStart] = []*BucketWrap{w}
		}
	}
	items := make([]*base.MetricItem, 0, 8)
	for ts, values := range wm {
		if len(values) == 0 {
			continue
		}
		if item := m.metricItemFromBuckets(ts, values); item != nil {
			items = append(items, item)
		}
	}
	return items
}

// metricItemFromBuckets aggregates multiple bucket wrappers (based on the same startTime in second)
// to the single MetricItem.
func (m *SlidingWindowMetric) metricItemFromBuckets(ts uint64, ws []*BucketWrap) *base.MetricItem {
	item := &base.MetricItem{Timestamp: ts}
	var allRt int64 = 0
	for _, w := range ws {
		mi := w.Value.Load()
		if mi == nil {
			logging.Error(errors.New("nil BucketWrap"), "Current bucket value is nil in SlidingWindowMetric.metricItemFromBuckets()")
			return nil
		}
		mb, ok := mi.(*MetricBucket)
		if !ok {
			logging.Error(errors.New("type assert failed"), "Fail to do type assert in SlidingWindowMetric.metricItemFromBuckets()", "bucketStartTime", w.BucketStart, "expectType", "*MetricBucket", "actualType", reflect.TypeOf(mb).Name())
			return nil
		}
		item.PassQps += uint64(mb.Get(base.MetricEventPass))
		item.BlockQps += uint64(mb.Get(base.MetricEventBlock))
		item.ErrorQps += uint64(mb.Get(base.MetricEventError))
		item.CompleteQps += uint64(mb.Get(base.MetricEventComplete))
		item.MonitorBlockQps += uint64(mb.Get(base.MetricEventMonitorBlock))
		allRt += mb.Get(base.MetricEventRt)
	}
	if item.CompleteQps > 0 {
		item.AvgRt = uint64(allRt) / item.CompleteQps
	} else {
		item.AvgRt = uint64(allRt)
	}
	return item
}

func (m *SlidingWindowMetric) metricItemFromBucket(w *BucketWrap) *base.MetricItem {
	mi := w.Value.Load()
	if mi == nil {
		logging.Error(errors.New("nil BucketWrap"), "Current bucket value is nil in SlidingWindowMetric.metricItemFromBucket()")
		return nil
	}
	mb, ok := mi.(*MetricBucket)
	if !ok {
		logging.Error(errors.New("type assert failed"), "Fail to do type assert in SlidingWindowMetric.metricItemFromBucket()", "expectType", "*MetricBucket", "actualType", reflect.TypeOf(mb).Name())
		return nil
	}
	completeQps := mb.Get(base.MetricEventComplete)
	item := &base.MetricItem{
		PassQps:         uint64(mb.Get(base.MetricEventPass)),
		BlockQps:        uint64(mb.Get(base.MetricEventBlock)),
		MonitorBlockQps: uint64(mb.Get(base.MetricEventMonitorBlock)),
		ErrorQps:        uint64(mb.Get(base.MetricEventError)),
		CompleteQps:     uint64(completeQps),
		Timestamp:       w.BucketStart,
	}
	if completeQps > 0 {
		item.AvgRt = uint64(mb.Get(base.MetricEventRt) / completeQps)
	} else {
		item.AvgRt = uint64(mb.Get(base.MetricEventRt))
	}
	return item
}

func (m *SlidingWindowMetric) MaxConcurrency() int64 {
	now := util.CurrentTimeMillis()
	start, end := m.getBucketStartRange(now)
	satisfiedBuckets := m.real.ValuesConditional(now, func(ws uint64) bool {
		return ws >= start && ws <= end
	})
	var maxConcurrency int64
	for _, w := range satisfiedBuckets {
		mb := w.Value.Load()
		if mb == nil {
			logging.Error(errors.New("Illegal state: current bucket value is nil when calculating max concurrency"),
				"Illegal state: current bucket value is nil when calculating max concurrency")
			continue
		}
		counter, ok := mb.(*MetricBucket)
		if !ok {
			logging.Error(errors.New("failed to cast to type MetricBucket"),
				fmt.Sprintf("Failed to cast data value(%+v) to MetricBucket type", mb))
			continue
		}
		v := counter.MaxConcurrency()
		if v > maxConcurrency {
			maxConcurrency = v
		}
	}
	return maxConcurrency
}

func (m *SlidingWindowMetric) SecondMaxConcurrency() int64 {
	end := util.CurrentTimeMillis()
	start := end - 1000
	satisfiedBuckets := m.real.ValuesConditional(end, func(ws uint64) bool {
		return ws >= start && ws <= end
	})
	var maxConcurrency int64
	for _, w := range satisfiedBuckets {
		mb := w.Value.Load()
		if mb == nil {
			logging.Error(errors.New("Illegal state: current bucket value is nil when calculating max concurrency"),
				"Illegal state: current bucket value is nil when calculating max concurrency")
			continue
		}
		counter, ok := mb.(*MetricBucket)
		if !ok {
			logging.Error(errors.New("failed to cast to type MetricBucket"),
				fmt.Sprintf("Failed to cast data value(%+v) to MetricBucket type", mb))
			continue
		}
		v := counter.MaxConcurrency()
		if v > maxConcurrency {
			maxConcurrency = v
		}
	}
	return maxConcurrency
}
