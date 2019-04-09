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
	"strconv"
	"testing"
)

func TestHashSet_Alloc(t *testing.T) {
	zone := InitMetricsZone("TestNewSharedMetrics", 10*1024*1024)
	defer zone.Detach()

	entryCount := 1000
	for i := 0; i < entryCount; i++ {
		entry, err := defaultZone.alloc("testEntry" + strconv.Itoa(i))
		if err != nil {
			t.Error(err)
		}

		entry.value = int64(i + 1)
	}

	// re-alloc and re-access
	entry, err := defaultZone.alloc("testEntry0")
	if err != nil {
		t.Error(err)
	}

	if entry.value != 1 {
		t.Error("testEntry0 value not correct")
	}
}

func TestHashSet_Free(t *testing.T) {
	zone := InitMetricsZone("TestNewSharedMetrics", 10*1024*1024)
	defer zone.Detach()

	entryCount := 1000
	for i := 0; i < entryCount; i++ {
		entry, err := defaultZone.alloc("testEntry" + strconv.Itoa(i))
		if err != nil {
			t.Error(err)
		}

		entry.value = int64(i + 1)
	}

	// re-alloc , free,  re-access
	entry, err := defaultZone.alloc("testEntry433")
	if err != nil {
		t.Error(err)
	}

	defaultZone.free(entry)

	realloc, err := defaultZone.alloc("testEntry433")
	if entry != realloc {
		t.Error("free and reuse not expected")
	}

}

func BenchmarkHashSet_Free(b *testing.B) {
	zone := InitMetricsZone("TestNewSharedMetrics", 30*1024*1024)
	defer zone.Detach()

	entryCount := 200000
	var entries []*hashEntry
	for i := 0; i < entryCount; i++ {
		entry, err := defaultZone.alloc("testEntry" + strconv.Itoa(i))
		if err != nil {
			b.Error(err)
		}

		entry.value = int64(i + 1)
		entries = append(entries, entry)
	}
	b.ResetTimer()
	b.StartTimer()
	//  free
	for _, e := range entries {
		defaultZone.free(e)
	}
	b.StopTimer()
}
