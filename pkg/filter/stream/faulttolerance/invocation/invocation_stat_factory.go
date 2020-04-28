package invocation

import (
	"mosn.io/api"
	"sync"
)

type InvocationStatFactory struct {
	invocationStats *sync.Map
}

func (f *InvocationStatFactory) GetInvocationStat(host *api.HostInfo, dimension InvocationDimension) *InvocationStat {
	key := dimension.GetInvocationKey()
	if value, ok := f.invocationStats.Load(key); ok {
		return value.(*InvocationStat)
	} else {
		stat := NewInvocationStat(host, dimension)
		if value, ok := f.invocationStats.LoadOrStore(key, stat); ok {
			return value.(*InvocationStat)
		} else {
			//f.regulator.Regulate(stat)
			return value.(*InvocationStat)
		}
	}
}
