package faulttolerance

import (
	"sync"
)

type InvocationStatFactory struct {
	invocationStats *sync.Map
}

func (f *InvocationStatFactory) GetInvocationStat(dimension InvocationStatDimension) InvocationStat {
	key := dimension.GetKey()
	if value, ok := f.invocationStats.Load(key); ok {
		return value.(InvocationStat)
	} else {
		stat := f.newInvocationStat(dimension)
		value, _ := f.invocationStats.LoadOrStore(key, stat)
		return value.(InvocationStat)
	}
}

func (f *InvocationStatFactory) newInvocationStat(dimension InvocationStatDimension) *InvocationStat {
	return &InvocationStat{
		dimension: dimension,
	}
}
