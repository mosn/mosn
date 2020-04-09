package faulttolerance

import (
	"sync"
	"sync/atomic"
)

type HostStatusManager struct {
	dimensionCount *sync.Map
	source         *sync.Map
}

func NewHostStatusManager() *HostStatusManager {
	return &HostStatusManager{
		source: new(sync.Map),
	}
}

func (m *HostStatusManager) PutUnHealthyHost(dimension string, host string, maxHost uint64) {
	if value, ok := m.dimensionCount.Load(dimension); ok {
		count := value.(uint64)
		if count >= maxHost {
			return
		} else {
			if atomic.CompareAndSwapUint64(&count, count, count+1) {
				m.source.Store(host, true)
			}
		}
	}
}

func (m *HostStatusManager) IsUnHealthy(host string) bool {
	if _, ok := m.source.Load(host); ok {
		return true
	}
	return false
}
