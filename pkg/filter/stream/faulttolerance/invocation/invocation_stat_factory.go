package invocation

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/regulator"
	"sync"
)

func init() {
	invocationStatFactoryInstance = newInvocationStatFactory()
}

type InvocationStatFactory struct {
	invocationStats *sync.Map
	regulator       regulator.Regulator
}

var invocationStatFactoryInstance *InvocationStatFactory

func GetInvocationStatFactoryInstance() *InvocationStatFactory {
	return invocationStatFactoryInstance
}

func newInvocationStatFactory() *InvocationStatFactory {
	invocationStatFactory := &InvocationStatFactory{
		invocationStats: new(sync.Map),
		regulator:       regulator.NewDefaultRegulator(),
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
			//f.regulator.Regulate(stat)
			return value.(*InvocationStat)
		}
	}
}

func (f *InvocationStatFactory) ReleaseInvocationStat(key string) {
	f.invocationStats.Delete(key)
}
