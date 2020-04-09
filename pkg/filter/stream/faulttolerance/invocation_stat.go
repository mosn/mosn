package faulttolerance

import (
	v2 "mosn.io/mosn/pkg/config/v2"
	"sync/atomic"
)

type InvocationStat struct {
	dimension      InvocationDimension
	windowLeft     int64
	callCount      uint64
	exceptionCount uint64
	sign           int32
}

func (s *InvocationStat) GetSign() bool {
	return atomic.CompareAndSwapInt32(&s.sign, 0, 1)
}

func (s *InvocationStat) ReleaseSign() {
	atomic.StoreInt32(&s.sign, 0)
}

func (s *InvocationStat) init(now int64) {
	s.windowLeft = now
	atomic.StoreUint64(&s.exceptionCount, 1)
	atomic.StoreUint64(&s.callCount, 1)
}

func (s *InvocationStat) count(isException bool) {
	atomic.AddUint64(&s.callCount, 1)
	if isException {
		atomic.AddUint64(&s.exceptionCount, 1)
	}
}

func (s *InvocationStat) Call(isException bool, config *v2.FaultToleranceFilterConfig) bool {
	now := GetNowMS()
	windowRight := s.windowLeft + config.TimeWindow
	if now > windowRight {
		if s.GetSign() {
			defer func() {
				s.ReleaseSign()
			}()
			if atomic.LoadUint64(&s.callCount) < config.LeastWindowCount {
				s.init(now)
				return false
			}
			exceptionRate := DivideUint64(s.exceptionCount, s.callCount)
			if exceptionRate < config.ExceptionThreshold {
				s.init(now)
				return false
			} else {
				s.init(now)
				return true
			}
		}
	} else {
		s.count(isException)
	}
	return false
}

func (s *InvocationStat) GetCallCount() uint64 {
	return s.callCount
}

func (s *InvocationStat) GetExceptionCount() uint64 {
	return s.exceptionCount
}
