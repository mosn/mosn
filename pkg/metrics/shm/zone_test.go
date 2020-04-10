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
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"mosn.io/mosn/pkg/types"
)

func TestNewSharedMetrics(t *testing.T) {
	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()
	zone := InitMetricsZone("TestNewSharedMetrics", 10*1024*1024)
	defer func() {
		zone.Detach()
		Reset()
	}()

	cpu := runtime.NumCPU()
	expected := cpu * 1000
	counter := 0

	wg := sync.WaitGroup{}
	wg.Add(cpu)

	for i := 0; i < cpu; i++ {
		go func() {
			for j := 0; j < 1000; j++ {
				zone.lock()
				counter++
				zone.unlock()
			}
			wg.Done()
		}()
	}

	wg.Wait()

	if expected != counter {
		t.Error("metrics zone lock & unlock not work")
	}
}

func TestClear(t *testing.T) {
	defer Reset()

	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()
	zone := InitMetricsZone("TestClear", 10*1024*1024)

	// 1. modify and detach, but do not delete the file
	entry, err := zone.alloc("TestGauge")
	if err != nil {
		t.Error(err)
	}

	gauge := ShmGauge(unsafe.Pointer(&entry.value))

	// update
	gauge.Update(5)

	// value
	if gauge.Value() != 5 {
		t.Error("gauge ops failed")
	}

	defer zone.Detach()

	// 2. attach without clear
	zone2 := createMetricsZone("TestClear", 10*1024*1024, false)

	entry, err = zone2.alloc("TestGauge")
	if err != nil {
		t.Error(err)
	}

	gauge = ShmGauge(unsafe.Pointer(&entry.value))

	// value
	if gauge.Value() != 5 {
		t.Error("gauge ops failed")
	}
	defer zone2.Detach()

	// 3. attach with clear

	zone3 := createMetricsZone("TestClear", 10*1024*1024, true)

	entry, err = zone3.alloc("TestGauge")
	if err != nil {
		t.Error(err)
	}

	gauge = ShmGauge(unsafe.Pointer(&entry.value))

	// value
	if gauge.Value() != 0 {
		t.Error("gauge ops failed")
	}
	defer zone3.Detach()
}
