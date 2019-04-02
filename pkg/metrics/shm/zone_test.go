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
)

func TestNewSharedMetrics(t *testing.T) {
	zone := InitMetricsZone("TestNewSharedMetrics", 10*1024*1024)
	defer zone.Detach()

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
