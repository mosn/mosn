package shm

import (
	"sync/atomic"
)

// metricsEntry is the mapping for metrics entry record memory-layout in shared memory.
//
// This struct should never be instantiated.
type metricsEntry struct {
	value int64     // 8
	ref   uint32    // 4
	name  [100]byte // 100
}

func (e *metricsEntry) assignName(name []byte) {
	i := 0
	for i = range name {
		e.name[i] = name[i]
	}
	e.name[i+1] = 0
}

func (e *metricsEntry) equalName(name []byte) bool {
	i := 0
	for i = range name {
		if e.name[i] != name[i] {
			return false
		}
	}
	// no more characters
	return e.name[i+1] == 0
}

func (e *metricsEntry) getName() []byte {
	for i := 0; i < len(e.name); i++ {
		if e.name[i] == 0 {
			return e.name[:i]
		}
	}
	return e.name[:]
}

func (e *metricsEntry) incRef() {
	atomic.AddUint32(&e.ref, 1)
}

func (e *metricsEntry) decRef() bool {
	return atomic.AddUint32(&e.ref, ^uint32(0)) == 0
}
