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
	"time"

	"golang.org/x/sync/errgroup"
	"gopkg.in/errgo.v2/fmt/errors"

	"github.com/alibaba/sentinel-golang/core/base"
)

const (
	RuleCheckSlotName  = "sentinel-core-hotspot-rule-check-slot"
	RuleCheckSlotOrder = 4000
)

var (
	DefaultSlot = &Slot{}
)

type Slot struct {
}

func (s *Slot) Name() string {
	return RuleCheckSlotName
}

func (s *Slot) Order() uint32 {
	return RuleCheckSlotOrder
}

func (s *Slot) Check(ctx *base.EntryContext) *base.TokenResult {
	res := ctx.Resource.Name()
	batch := int64(ctx.Input.BatchCount)
	g, _ := errgroup.WithContext(context.Background())

	result := ctx.RuleCheckResult
	tcs := getTrafficControllersFor(res)
	for _, tc := range tcs {
		args := tc.ExtractArgs(ctx)
		if args == nil || len(args) == 0 {
			continue
		}

		for _, arg := range args {
			arg := arg // https://golang.org/doc/faq#closures_and_goroutines
			g.Go(func() error {
				r := canPassCheck(tc, arg, batch)
				if r == nil {
					return nil
				}

				if r.Status() == base.ResultStatusBlocked {
					if tc.BoundRule().Mode == MONITOR {
						PutOutputAttachment(ctx, KeyIsMonitorBlocked, true)
						return nil
					}
					return errors.Newf("concurrent canPassCheck err=%v", r.BlockError())
				}
				if r.Status() == base.ResultStatusShouldWait {
					if nanosToWait := r.NanosToWait(); nanosToWait > 0 {
						// Handle waiting action.
						time.Sleep(nanosToWait)
					}
					return nil
				}
				return nil
			})
		}
	}
	if err := g.Wait(); err != nil {
		result.ResetToBlockedWithMessage(base.BlockTypeHotSpotParamFlow, err.Error())
		return result
	}
	return result
}

func canPassCheck(tc TrafficShapingController, arg interface{}, batch int64) *base.TokenResult {
	return canPassLocalCheck(tc, arg, batch)
}

func canPassLocalCheck(tc TrafficShapingController, arg interface{}, batch int64) *base.TokenResult {
	return tc.PerformChecking(arg, batch)
}

const KeyIsMonitorBlocked = "isMonitorBlocked"

func PutOutputAttachment(ctx *base.EntryContext, key interface{}, value interface{}) {
	if ctx.Data == nil {
		ctx.Data = make(map[interface{}]interface{})
	}
	ctx.Data[key] = value
}
