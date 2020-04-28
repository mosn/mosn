package invocation

import (
	"mosn.io/api"
)

type InvocationStat struct {
	dimension      InvocationDimension
	host           *api.HostInfo
	callCount      int64
	exceptionCount int64
	uselessCycle   int32
	healthy        bool
	downgradeTime  int64
}

func NewInvocationStat(host *api.HostInfo, dimension InvocationDimension) *InvocationStat {
	invocationStat := &InvocationStat{
		dimension:      dimension,
		host:           host,
		callCount:      0,
		exceptionCount: 0,
		uselessCycle:   0,
		healthy:        false,
		downgradeTime:  0,
	}
	return invocationStat
}
