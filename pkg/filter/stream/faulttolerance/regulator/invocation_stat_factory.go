package regulator

import (
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"sync"
)

type InvocationStatFactory struct {
	invocationStats *sync.Map
	regulator       Regulator
}

var invocationStatFactoryInstance *InvocationStatFactory

func GetInvocationStatFactoryInstance() *InvocationStatFactory {
	return invocationStatFactoryInstance
}

func NewInvocationStatFactory(config *v2.FaultToleranceFilterConfig) *InvocationStatFactory {
	invocationStatFactory := &InvocationStatFactory{
		invocationStats: new(sync.Map),
		regulator:       NewDefaultRegulator(config),
	}
	return invocationStatFactory
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
			f.regulator.Regulate(stat)
			return value.(*InvocationStat)
		}
	}
}

func (f *InvocationStatFactory) ReleaseInvocationStat(key string) {
	f.invocationStats.Delete(key)
}
