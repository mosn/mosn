package faulttolerance

import (
	"sync"
)

type InvocationStatFactory struct {
	invocationStats *sync.Map
}

var InvocationStatFactoryInstance *InvocationStatFactory

func GetInvocationStatFactoryInstance() *InvocationStatFactory {
	return InvocationStatFactoryInstance
}

func (f *InvocationStatFactory) GetInvocationStat(dimension InvocationStatDimension) InvocationStat {
	key := dimension.GetInvocationKey()
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

func (f *InvocationStatFactory) getSnapshotInvocationStats(dimensions *sync.Map) []*InvocationStat {
	invocationStats := []*InvocationStat{}
	dimensions.Range(func(key, value interface{}) bool {
		dimension := value.(InvocationStatDimension)
		invocationStat := f.GetInvocationStat(dimension)
		invocationStats = append(invocationStats, invocationStat.Snapshot())
		return true
	})

	return invocationStats
}
