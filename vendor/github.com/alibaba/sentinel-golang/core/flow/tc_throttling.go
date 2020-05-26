package flow

import (
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/util"
	"math"
	"sync/atomic"
	"time"
)

const nanoUnitOffset = time.Second / time.Nanosecond

// ThrottlingChecker limits the time interval between two requests.
type ThrottlingChecker struct {
	maxQueueingTimeNs uint64

	lastPassedTime uint64

	// TODO: support strict mode
	strict bool
}

func NewThrottlingChecker(timeoutMs uint32) *ThrottlingChecker {
	return &ThrottlingChecker{
		maxQueueingTimeNs: uint64(timeoutMs) * util.UnixTimeUnitOffset,
		lastPassedTime:    0,
	}
}

func (c *ThrottlingChecker) DoCheck(_ base.StatNode, acquireCount uint32, threshold float64) *base.TokenResult {
	// Pass when acquire count is less or equal than 0.
	if acquireCount <= 0 {
		return base.NewTokenResultPass()
	}
	if threshold <= 0 {
		return base.NewTokenResultBlocked(base.BlockTypeFlow, "Flow")
	}
	// Here we use nanosecond so that we could control the queueing time more accurately.
	curNano := util.CurrentTimeNano()
	// The interval between two requests (in nanoseconds).
	interval := uint64(math.Ceil(float64(acquireCount) / threshold * float64(nanoUnitOffset)))

	// Expected pass time of this request.
	expectedTime := atomic.LoadUint64(&c.lastPassedTime) + interval
	if expectedTime <= curNano {
		// Contention may exist here, but it's okay.
		atomic.StoreUint64(&c.lastPassedTime, curNano)
		return base.NewTokenResultPass()
	}
	estimatedQueueingDuration := atomic.LoadUint64(&c.lastPassedTime) + interval - util.CurrentTimeNano()
	if estimatedQueueingDuration > c.maxQueueingTimeNs {
		return base.NewTokenResultBlocked(base.BlockTypeFlow, "Flow")
	}

	oldTime := atomic.AddUint64(&c.lastPassedTime, interval)
	estimatedQueueingDuration = oldTime - util.CurrentTimeNano()
	if estimatedQueueingDuration > c.maxQueueingTimeNs {
		// Subtract the interval.
		atomic.AddUint64(&c.lastPassedTime, ^(interval - 1))
		return base.NewTokenResultBlocked(base.BlockTypeFlow, "Flow")
	}
	if estimatedQueueingDuration > 0 {
		return base.NewTokenResultShouldWait(estimatedQueueingDuration / util.UnixTimeUnitOffset)
	} else {
		return base.NewTokenResultShouldWait(0)
	}
}
