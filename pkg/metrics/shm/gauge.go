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
		if defaultZone != nil {
			if entry, err := defaultZone.alloc(name); err == nil {
				return ShmGauge(unsafe.Pointer(&entry.value))
			}
		} else if fallback {
			return gometrics.NewGauge()
		}
		return gometrics.NilGauge{}
	}
}

// stoppable
func (c ShmGauge) Stop() {
	if defaultZone != nil {
		defaultZone.free((*hashEntry)(unsafe.Pointer(c)))
	}
}
