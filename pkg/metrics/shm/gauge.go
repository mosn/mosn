package shm

import (
	"sync/atomic"
	gometrics "github.com/rcrowley/go-metrics"
	"unsafe"
)

// StandardGauge is the standard implementation of a Gauge and uses the
// sync/atomic package to manage a single int64 value.
type ShmGauge uintptr

// Snapshot returns a read-only copy of the gauge.
func (g ShmGauge) Snapshot() gometrics.Gauge {
	return gometrics.GaugeSnapshot(g.Value())
}

// Update updates the gauge's value.
func (g ShmGauge) Update(v int64) {
	atomic.StoreInt64((*int64)(unsafe.Pointer(g)), v)
}

// Value returns the gauge's current value.
func (g ShmGauge) Value() int64 {
	return atomic.LoadInt64((*int64)(unsafe.Pointer(g)))
}

func NewShmGaugeFunc(name string) func() gometrics.Gauge {
	return func() gometrics.Gauge {
		entry, err := defaultZone.alloc(name)
		if err != nil {
			return gometrics.NilGauge{}
		}

		return ShmGauge(unsafe.Pointer(&entry.value))
	}
}

// stoppable
func (c ShmGauge) Stop() {
	defaultZone.free((*hashEntry)(unsafe.Pointer(c)))
}