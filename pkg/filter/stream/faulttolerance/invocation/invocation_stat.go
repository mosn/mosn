package invocation

import (
	"mosn.io/api"
	"sync/atomic"
)

type InvocationStat struct {
	dimension      InvocationDimension
	host           *api.HostInfo
	callCount      int64
	exceptionCount int64
	uselessCycle   int32
	downgradeTime  int64
}

func NewInvocationStat(host *api.HostInfo, dimension InvocationDimension) *InvocationStat {
	invocationStat := &InvocationStat{
		dimension:      dimension,
		host:           host,
		callCount:      0,
		exceptionCount: 0,
		uselessCycle:   0,
		downgradeTime:  0,
	}
	return invocationStat
}

func (s *InvocationStat) Call(isException bool) {
	atomic.AddInt64(&s.callCount, 1)
	if isException {
		atomic.AddInt64(&s.exceptionCount, 1)
	}
}
