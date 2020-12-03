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
	"errors"
	"os"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/shm"
)

var (
	pageSize = 4 * 1024
	pid      = uint32(os.Getpid())

	defaultZone *zone
)

// InitDefaultMetricsZone used to initialize the default zone according to the configuration.
func InitDefaultMetricsZone(name string, size int, clear bool) {
	zone := createMetricsZone(name, size, clear)
	defaultZone = zone
}

// InitMetricsZone used to initialize the default zone according to the configuration.
// It's caller's responsibility to detach the zone.
func InitMetricsZone(name string, size int) *zone {
	defaultZone = createMetricsZone(name, size, false)
	return defaultZone
}

func Reset() {
	defaultZone = nil
}

// createMetricsZone used to create new shm-based metrics zone. It's caller's responsibility
// to detach the zone.
func createMetricsZone(name string, size int, clear bool) *zone {
	if clear {
		shm.Clear(name)
	}

	zone, err := newSharedMetrics(name, size)
	if err != nil {
		log.DefaultLogger.Fatalf("open shared memory for metrics failed: %v", err)
	}
	return zone
}

// zone is the in-heap struct that holds the reference to the entire metrics shared memory.
// ATTENTION: entries is modified so that it points to the shared memory entries address.
type zone struct {
	span *shm.ShmSpan

	mutex *uint32
	ref   *uint32

	set *hashSet // mutex + ref = 64bit, so atomic ops has no problem
}

func newSharedMetrics(name string, size int) (*zone, error) {
	alignedSize := align(size, pageSize)

	span, err := shm.Alloc(name, alignedSize)
	if err != nil {
		return nil, err
	}
	// 1. mutex and ref
	mutex, err := span.Alloc(4)
	if err != nil {
		return nil, err
	}

	ref, err := span.Alloc(4)
	if err != nil {
		return nil, err
	}

	zone := &zone{
		span:  span,
		mutex: (*uint32)(unsafe.Pointer(mutex)),
		ref:   (*uint32)(unsafe.Pointer(ref)),
	}

	// 2. hashSet

	// assuming that 100 entries with 50 slots, so the ratio of occupied memory is
	// entries:slots  = 100 x 128 : 50 x 4 = 64 : 1
	// so assuming slots memory size is N, total allocated memory size is M, then we have:
	// M - 1024 < 65N + 28 <= M

	slotsNum := (alignedSize - 28) / (65 * 4)
	slotsSize := slotsNum * 4
	entryNum := slotsNum * 2
	entrySize := slotsSize * 64

	hashSegSize := entrySize + 20 + slotsSize
	hashSegment, err := span.Alloc(hashSegSize)
	if err != nil {
		return nil, err
	}

	// if zones's ref > 0, no need to initialize hashset's value
	set, err := newHashSet(hashSegment, hashSegSize, entryNum, slotsNum, atomic.LoadUint32(zone.ref) == 0)
	if err != nil {
		return nil, err
	}
	zone.set = set

	// add ref
	atomic.AddUint32(zone.ref, 1)

	return zone, nil
}

func (z *zone) lock() {
	times := 0

	// 5ms spin interval, 5 times burst
	for {
		if atomic.CompareAndSwapUint32(z.mutex, 0, pid) {
			return
		}

		time.Sleep(time.Millisecond)
		times++

		if times%5 == 0 {
			// check the lock holder, if it is not current process, force unlock and update the holder.
			if atomic.LoadUint32(z.mutex) != pid {
				atomic.StoreUint32(z.mutex, pid)
				return
			}
			runtime.Gosched()
		}
	}
}

func (z *zone) unlock() {
	if !atomic.CompareAndSwapUint32(z.mutex, pid, 0) {
		log.DefaultLogger.Alertf("metrics.shm", "[metrics][shm] unexpected lock holder, can not unlock")
	}

}

func (z *zone) alloc(name string) (*hashEntry, error) {
	z.lock()
	defer z.unlock()

	entry, create := z.set.Alloc(name)
	if entry == nil {
		// TODO log & stat
		return nil, errors.New("alloc failed")
	}

	// for existed entry, increase its reference
	if !create {
		entry.incRef()
	}

	return entry, nil
}

func (z *zone) free(entry *hashEntry) error {
	z.lock()
	defer z.unlock()

	z.set.Free(entry)
	return nil
}

func (z *zone) Detach() {
	// ensure all process detached
	if atomic.AddUint32(z.ref, ^uint32(0)) == 0 {
		shm.Free(z.span)
	}
}

func align(size, alignment int) int {
	return (size + alignment - 1) & ^(alignment - 1)
}
