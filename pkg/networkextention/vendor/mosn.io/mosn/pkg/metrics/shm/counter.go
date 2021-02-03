/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package shm

import (
	"sync/atomic"
	"unsafe"

	gometrics "github.com/rcrowley/go-metrics"
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

func NewShmCounterFunc(name string) func() gometrics.Counter {
	return func() gometrics.Counter {
		if defaultZone != nil {
			if entry, err := defaultZone.alloc(name); err == nil {
				return ShmCounter(unsafe.Pointer(&entry.value))
			}
		} else if fallback {
			return gometrics.NewCounter()
		}
		return gometrics.NilCounter{}
	}
}

// stoppable
func (c ShmCounter) Stop() {
	if defaultZone != nil {
		defaultZone.free((*hashEntry)(unsafe.Pointer(c)))
	}
}
