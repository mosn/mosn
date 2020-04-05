package faulttolerance

import (
	"sync/atomic"
)

type InvocationStat struct {
	dimension      InvocationStatDimension
	callCount      *uint64
	exceptionCount *uint64
}

func (s *InvocationStat) Call(isException bool) {
	atomic.AddUint64(s.callCount, 1)
	if isException {
		atomic.AddUint64(s.exceptionCount, 1)
	}
}

func (s *InvocationStat) GetCallCount() uint64 {
	return *s.callCount
}

func (s *InvocationStat) GetExceptionCount() uint64 {
	return *s.exceptionCount
}

func (s *InvocationStat) Snapshot() *InvocationStat {
	return &InvocationStat{
		dimension:      s.dimension,
		callCount:      s.callCount,
		exceptionCount: s.exceptionCount,
	}
}
