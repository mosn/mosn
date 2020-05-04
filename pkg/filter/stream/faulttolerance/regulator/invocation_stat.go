package regulator

import (
	"fmt"
	"mosn.io/api"
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/util"
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

func (s *InvocationStat) GetMeasureKey() string {
	return s.GetMeasureKey()
}

func (s *InvocationStat) GetInvocationKey() string {
	return s.GetInvocationKey()
}

func (s *InvocationStat) GetCall() int64 {
	return s.callCount
}

func (s *InvocationStat) GetCount() (int64, int64) {
	return s.callCount, s.exceptionCount
}

func (s *InvocationStat) GetDowngradeTime() int64 {
	return s.downgradeTime
}

func (s *InvocationStat) AddUselessCycle() bool {
	s.uselessCycle++
	return s.uselessCycle >= 6
}

func (s *InvocationStat) RestUselessCycle() {
	s.uselessCycle = 0
}

func (s *InvocationStat) GetExceptionRate() (bool, float64) {
	if s.callCount <= 0 {
		return false, 0
	}
	return true, util.DivideInt64(s.exceptionCount, s.callCount)
}

func (s *InvocationStat) Snapshot() *InvocationStat {
	return &InvocationStat{
		dimension:      s.dimension,
		host:           s.host,
		callCount:      s.callCount,
		exceptionCount: s.exceptionCount,
		uselessCycle:   s.uselessCycle,
		downgradeTime:  s.downgradeTime,
	}
}

func (s *InvocationStat) Update(snapshot *InvocationStat) {
	call, exception := snapshot.GetCount()
	atomic.AddInt64(&s.exceptionCount, -exception)
	atomic.AddInt64(&s.callCount, -call)
}

func (s *InvocationStat) Downgrade() {
	(*s.host).SetHealthFlag(api.FAILED_OUTLIER_CHECK)
}

func (s *InvocationStat) Recover() {
	s.exceptionCount = 0
	s.callCount = 0
	s.uselessCycle = 0
	s.downgradeTime = 0
	if s.host != nil {
		(*s.host).ClearHealthFlag(api.FAILED_OUTLIER_CHECK)
	}
}

func (s *InvocationStat) IsHealthy() bool {
	return (*s.host).Health()
}

func (s *InvocationStat) String() string {
	str := fmt.Sprintf("host=%s, dimension=%s,callCount=%v,exceptionCount=%v,uselessCycle=%v, downgradeTime=%v",
		(*s.host).AddressString(), s.dimension, s.callCount, s.exceptionCount, s.uselessCycle, s.downgradeTime)
	return str
}
