// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

package gxsync

import (
	"sync/atomic"
)

//type TryLock struct {
//	lock int32
//}
//
//func (l *TryLock) Lock() bool {
//	return atomic.CompareAndSwapInt32(&(l.lock), 0, 1)
//}
//
//func (l *TryLock) Unlock() {
//	atomic.StoreInt32(&(l.lock), 0)
//}

// ref: https://github.com/LK4D4/trylock/blob/master/trylock.go
import (
	"sync"
	"unsafe"
)

const mutexLocked = 1 << iota

// Mutex is simple sync.Mutex + ability to try to Lock.
type Mutex struct {
	lock sync.Mutex
}

// Lock locks m.
// If the lock is already in use, the calling goroutine
// blocks until the mutex is available.
func (m *Mutex) Lock() {
	m.lock.Lock()
}

// Unlock unlocks m.
// It is a run-time error if m is not locked on entry to Unlock.
//
// A locked Mutex is not associated with a particular goroutine.
// It is allowed for one goroutine to lock a Mutex and then
// arrange for another goroutine to unlock it.
func (m *Mutex) Unlock() {
	m.lock.Unlock()
}

// TryLock tries to lock m. It returns true in case of success, false otherwise.
func (m *Mutex) TryLock() bool {
	return atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&m.lock)), 0, mutexLocked)
}

// IsLocked returns the lock state
func (m *Mutex) IsLocked() bool {
	return atomic.LoadInt32((*int32)(unsafe.Pointer(&m.lock))) == mutexLocked
}
