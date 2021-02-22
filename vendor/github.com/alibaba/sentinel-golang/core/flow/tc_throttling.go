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

package flow

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/util"
)

const (
	BlockMsgQueueing = "flow throttling check blocked, estimated queueing time exceeds max queueing time"

	MillisToNanosOffset = int64(time.Millisecond / time.Nanosecond)
)

// ThrottlingChecker limits the time interval between two requests.
type ThrottlingChecker struct {
	owner             *TrafficShapingController
	maxQueueingTimeNs int64
	statIntervalNs    int64
	lastPassedTime    int64
}

func NewThrottlingChecker(owner *TrafficShapingController, timeoutMs uint32, statIntervalMs uint32) *ThrottlingChecker {
	var statIntervalNs int64
	if statIntervalMs == 0 {
		statIntervalNs = 1000 * MillisToNanosOffset
	} else {
		statIntervalNs = int64(statIntervalMs) * MillisToNanosOffset
	}
	return &ThrottlingChecker{
		owner:             owner,
		maxQueueingTimeNs: int64(timeoutMs) * MillisToNanosOffset,
		statIntervalNs:    statIntervalNs,
		lastPassedTime:    0,
	}
}
func (c *ThrottlingChecker) BoundOwner() *TrafficShapingController {
	return c.owner
}

func (c *ThrottlingChecker) DoCheck(_ base.StatNode, batchCount uint32, threshold float64) *base.TokenResult {
	// Pass when batch count is less or equal than 0.
	if batchCount <= 0 {
		return nil
	}

	var rule *Rule
	if c.BoundOwner() != nil {
		rule = c.BoundOwner().BoundRule()
	}

	if threshold <= 0.0 {
		msg := "flow throttling check blocked, threshold is <= 0.0"
		return base.NewTokenResultBlockedWithCause(base.BlockTypeFlow, msg, rule, nil)
	}
	if float64(batchCount) > threshold {
		return base.NewTokenResultBlocked(base.BlockTypeFlow)
	}
	// Here we use nanosecond so that we could control the queueing time more accurately.
	curNano := int64(util.CurrentTimeNano())

	// The interval between two requests (in nanoseconds).
	intervalNs := int64(math.Ceil(float64(batchCount) / threshold * float64(c.statIntervalNs)))

	// Expected pass time of this request.
	expectedTime := atomic.LoadInt64(&c.lastPassedTime) + intervalNs
	if expectedTime <= curNano {
		// Contention may exist here, but it's okay.
		atomic.StoreInt64(&c.lastPassedTime, curNano)
		return nil
	}

	estimatedQueueingDuration := atomic.LoadInt64(&c.lastPassedTime) + intervalNs - int64(util.CurrentTimeNano())
	if estimatedQueueingDuration > c.maxQueueingTimeNs {
		return base.NewTokenResultBlockedWithCause(base.BlockTypeFlow, BlockMsgQueueing, rule, nil)
	}

	oldTime := atomic.AddInt64(&c.lastPassedTime, intervalNs)
	estimatedQueueingDuration = oldTime - int64(util.CurrentTimeNano())
	if estimatedQueueingDuration > c.maxQueueingTimeNs {
		// Subtract the interval.
		atomic.AddInt64(&c.lastPassedTime, -intervalNs)
		return base.NewTokenResultBlockedWithCause(base.BlockTypeFlow, BlockMsgQueueing, rule, nil)
	}
	if estimatedQueueingDuration > 0 {
		return base.NewTokenResultShouldWait(time.Duration(estimatedQueueingDuration))
	} else {
		return base.NewTokenResultShouldWait(0)
	}
}
