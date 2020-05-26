package base

import (
	"fmt"
	"sync/atomic"

	"github.com/alibaba/sentinel-golang/core/base"
)

// MetricBucket represents the entity to record metrics per minimum time unit (i.e. the bucket time span).
// Note that all operations of the MetricBucket are required to be thread-safe.
type MetricBucket struct {
	// value of statistic
	counter        [base.MetricEventTotal]int64
	minRt          int64
	maxConcurrency int64
}

func NewMetricBucket() *MetricBucket {
	mb := &MetricBucket{
		minRt: base.DefaultStatisticMaxRt,
	}
	return mb
}

// Add statistic count for the given metric event.
func (mb *MetricBucket) Add(event base.MetricEvent, count int64) {
	if event >= base.MetricEventTotal || event < 0 {
		panic(fmt.Sprintf("Unknown metric event: %v", event))
	}
	if event == base.MetricEventRt {
		mb.AddRt(count)
		return
	}
	mb.addCount(event, count)
}

func (mb *MetricBucket) addCount(event base.MetricEvent, count int64) {
	atomic.AddInt64(&mb.counter[event], count)
}

// Get current statistic count of the given metric event.
func (mb *MetricBucket) Get(event base.MetricEvent) int64 {
	if event >= base.MetricEventTotal || event < 0 {
		panic(fmt.Sprintf("Unknown metric event: %v", event))
	}
	return atomic.LoadInt64(&mb.counter[event])
}

func (mb *MetricBucket) reset() {
	for i := 0; i < int(base.MetricEventTotal); i++ {
		atomic.StoreInt64(&mb.counter[i], 0)
	}
	atomic.StoreInt64(&mb.minRt, base.DefaultStatisticMaxRt)
}

func (mb *MetricBucket) AddRt(rt int64) {
	mb.addCount(base.MetricEventRt, rt)
	if rt < atomic.LoadInt64(&mb.minRt) {
		atomic.StoreInt64(&mb.minRt, rt)
	}
}

func (mb *MetricBucket) MinRt() int64 {
	return atomic.LoadInt64(&mb.minRt)
}

func (mb *MetricBucket) UpdateMaxConcurrency(count int64) {
	if count > atomic.LoadInt64(&mb.maxConcurrency) {
		atomic.StoreInt64(&mb.maxConcurrency, count)
	}
}

func (mb *MetricBucket) MaxConcurrency() int64 {
	return atomic.LoadInt64(&mb.maxConcurrency)
}
