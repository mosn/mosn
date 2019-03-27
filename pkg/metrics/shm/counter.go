package shm

import (
	"sync/atomic"
	gometrics "github.com/rcrowley/go-metrics"
	"unsafe"
)

// StandardCounter is the standard implementation of a Counter and uses the
// sync/atomic package to manage a single int64 value.
type ShmCounter uintptr

// Clear sets the counter to zero.
func (c ShmCounter) Clear() {
	atomic.StoreInt64((*int64)(unsafe.Pointer(c)), 0)
}

// Count returns the current count.
func (c ShmCounter) Count() int64 {
	return atomic.LoadInt64((*int64)(unsafe.Pointer(c)))
}

// Dec decrements the counter by the given amount.
func (c ShmCounter) Dec(i int64) {
	atomic.AddInt64((*int64)(unsafe.Pointer(c)), -i)
}

// Inc increments the counter by the given amount.
func (c ShmCounter) Inc(i int64) {
	atomic.AddInt64((*int64)(unsafe.Pointer(c)), i)
}

// Snapshot returns a read-only copy of the counter.
func (c ShmCounter) Snapshot() gometrics.Counter {
	return gometrics.CounterSnapshot(c.Count())
}

func NewShmCounter(entry *metricsEntry) gometrics.Counter {
	return ShmCounter(unsafe.Pointer(&entry.value))
}
