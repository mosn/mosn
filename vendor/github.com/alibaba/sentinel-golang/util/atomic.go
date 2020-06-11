package util

import "sync/atomic"

func IncrementAndGetInt64(v *int64) int64 {
	old := int64(0)
	for {
		old = atomic.LoadInt64(v)
		if atomic.CompareAndSwapInt64(v, old, old+1) {
			break
		}
	}
	return old + 1
}

type AtomicBool struct {
	// default 0, means false
	flag int32
}

func (b *AtomicBool) CompareAndSet(old, new bool) bool {
	if old == new {
		return true
	}
	var oldInt, newInt int32
	if old {
		oldInt = 1
	}
	if new {
		newInt = 1
	}
	return atomic.CompareAndSwapInt32(&(b.flag), oldInt, newInt)
}

func (b *AtomicBool) Set(value bool) {
	i := int32(0)
	if value {
		i = 1
	}
	atomic.StoreInt32(&(b.flag), int32(i))
}

func (b *AtomicBool) Get() bool {
	if atomic.LoadInt32(&(b.flag)) != 0 {
		return true
	}
	return false
}
