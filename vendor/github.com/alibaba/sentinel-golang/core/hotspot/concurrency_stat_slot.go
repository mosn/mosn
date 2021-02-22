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

package hotspot

import (
	"context"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/logging"
)

const (
	StatSlotName  = "sentinel-core-hotspot-concurrency-stat-slot"
	StatSlotOrder = 4000
)

var (
	DefaultConcurrencyStatSlot = &ConcurrencyStatSlot{}
)

// ConcurrencyStatSlot is to record the Concurrency statistic for all arguments
type ConcurrencyStatSlot struct {
}

func (c *ConcurrencyStatSlot) Name() string {
	return StatSlotName
}

func (s *ConcurrencyStatSlot) Order() uint32 {
	return StatSlotOrder
}

func (c *ConcurrencyStatSlot) OnEntryPassed(ctx *base.EntryContext) {
	res := ctx.Resource.Name()
	g, _ := errgroup.WithContext(context.Background())
	tcs := getTrafficControllersFor(res)

	for _, tc := range tcs {
		if tc.BoundRule().MetricType != Concurrency {
			continue
		}
		args := tc.ExtractArgs(ctx)
		if args == nil || len(args) == 0 {
			continue
		}
		for _, arg := range args {
			arg := arg // https://golang.org/doc/faq#closures_and_goroutines
			g.Go(func() error {
				metric := tc.BoundMetric()
				concurrencyPtr, existed := metric.ConcurrencyCounter.Get(arg)
				if !existed || concurrencyPtr == nil {
					if logging.DebugEnabled() {
						logging.Debug("[ConcurrencyStatSlot OnEntryPassed] Parameter does not exist in ConcurrencyCounter.", "argument", arg)
					}
					return nil
				}
				atomic.AddInt64(concurrencyPtr, 1)
				return nil
			})
		}
	}
	g.Wait()
	return
}

func (c *ConcurrencyStatSlot) OnEntryBlocked(ctx *base.EntryContext, blockError *base.BlockError) {
	// Do nothing
}

func (c *ConcurrencyStatSlot) OnCompleted(ctx *base.EntryContext) {
	res := ctx.Resource.Name()
	g, _ := errgroup.WithContext(context.Background())
	tcs := getTrafficControllersFor(res)

	for _, tc := range tcs {
		if tc.BoundRule().MetricType != Concurrency {
			continue
		}
		args := tc.ExtractArgs(ctx)
		if args == nil || len(args) == 0 {
			continue
		}
		for _, arg := range args {
			arg := arg // https://golang.org/doc/faq#closures_and_goroutines
			g.Go(func() error {
				metric := tc.BoundMetric()
				concurrencyPtr, existed := metric.ConcurrencyCounter.Get(arg)
				if !existed || concurrencyPtr == nil {
					if logging.DebugEnabled() {
						logging.Debug("[ConcurrencyStatSlot OnCompleted] Parameter does not exist in ConcurrencyCounter.", "argument", arg)
					}
					return nil
				}
				atomic.AddInt64(concurrencyPtr, -1)
				return nil
			})
		}
	}
	g.Wait()
	return
}
