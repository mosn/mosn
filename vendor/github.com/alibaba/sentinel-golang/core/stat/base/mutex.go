package base

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const mutexLocked = 1 << iota

// The mutex which supports try-locking.
type mutex struct {
	sync.Mutex
}

// TryLock acquires the lock only if it is free at the time of invocation.
func (tl *mutex) TryLock() bool {
	return atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&tl.Mutex)), 0, mutexLocked)
}
