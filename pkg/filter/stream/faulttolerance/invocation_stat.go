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
