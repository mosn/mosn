package faultinject

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"math/rand"
	"sync/atomic"
	"time"
)

type faultinjecter struct {
	// 1~100
	delayPercent  uint32
	delayDuration uint64
	delaying      uint32
	readCallbacks types.ReadFilterCallbacks
}

func NewFaultInjecter(config *v2.FaultInject) FaultInjecter {
	return &faultinjecter{
		delayPercent:  config.DelayPercent,
		delayDuration: config.DelayDuration,
	}
}

func (fi *faultinjecter) OnData(buffer types.IoBuffer) types.FilterStatus {
	fi.tryInjectDelay()

	if atomic.LoadUint32(&fi.delaying) > 0 {
		return types.StopIteration
	} else {
		return types.Continue
	}
}

func (fi *faultinjecter) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (fi *faultinjecter) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {
	fi.readCallbacks = cb
}

func (fi *faultinjecter) tryInjectDelay() {
	if atomic.LoadUint32(&fi.delaying) > 0 {
		return
	}

	duration := fi.getDelayDuration()

	if duration > 0 {
		if atomic.CompareAndSwapUint32(&fi.delaying, 0, 1) {
			go func() {
				select {
				case <-time.After(time.Duration(duration) * time.Millisecond):
					atomic.StoreUint32(&fi.delaying, 0)
					fi.readCallbacks.ContinueReading()
				}
			}()
		}
	}
}

func (fi *faultinjecter) getDelayDuration() uint64 {
	if fi.delayPercent == 0 {
		return 0
	}

	if uint32(rand.Intn(100))+1 > fi.delayPercent {
		return 0
	}

	return fi.delayDuration
}
