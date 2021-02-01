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
	"sync/atomic"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/pkg/errors"
)

// MetricBucket represents the entity to record metrics per minimum time unit (i.e. the bucket time span).
// Note that all operations of the MetricBucket are required to be thread-safe.
type MetricBucket struct {
	// Value of statistic
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
		logging.Error(errors.Errorf("Unknown metric event: %v", event), "")
		return
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
		logging.Error(errors.Errorf("Unknown metric event: %v", event), "")
		return 0
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
