package shm

import (
	"unsafe"
	"sync/atomic"
)

var (
	entrySize = int(unsafe.Sizeof(metricsEntry{}))
)

// metricsEntry is the mapping for metrics entry record memory-layout in shared memory.
// One entry is designed with 128 byte width, which is usually the cache-line size to avoid
// false sharing.
//
// This struct should never be instantiated.
type metricsEntry struct {
	name  [116]byte // 116
	value int64    // 8
	ref   uint32   // 4
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

func (e *metricsEntry) incRef() {
	atomic.AddUint32(&e.ref, 1)
}

func (e *metricsEntry) decRef() bool {
	return atomic.AddUint32(&e.ref, ^uint32(0)) == 0
}
