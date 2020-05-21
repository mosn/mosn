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
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"mosn.io/mosn/pkg/types"
)

func TestEntry(t *testing.T) {
	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()
	zone := InitMetricsZone("TestEntry", 10*1024)
	defer func() {
		zone.Detach()
		Reset()
	}()

	entry1, _ := defaultZone.alloc("entry_test_1")

	expected := 10000
	cpu := runtime.NumCPU()
	wg := sync.WaitGroup{}
	wg.Add(cpu * 2)
	for i := 0; i < cpu; i++ {
		go func() {
			for j := 0; j < expected/cpu; j++ {
				atomic.AddInt64(&entry1.value, 1)
			}
			wg.Done()
		}()
	}

	entry2, _ := defaultZone.alloc("entry_test_2")
	for i := 0; i < cpu; i++ {
		go func() {
			for j := 0; j < expected/cpu; j++ {
				atomic.AddInt64(&entry2.value, 1)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	fmt.Printf("entry1 %+v\n", entry1)
	fmt.Printf("entry2 %+v\n", entry2)

}

func BenchmarkMultiEntry(b *testing.B) {
	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()
	zone := InitMetricsZone("TestMultiEntry", 10*1024*1024)
	defer func() {
		zone.Detach()
		Reset()
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		defaultZone.alloc("bench_" + strconv.Itoa(i))
	}
}
