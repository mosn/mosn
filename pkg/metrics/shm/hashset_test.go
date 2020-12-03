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
	"math/rand"
	"strconv"
	"testing"
	"unsafe"

	"mosn.io/mosn/pkg/types"
)

func TestHashSet_Alloc(t *testing.T) {
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

func TestHashSet_EntrySize(t *testing.T) {
	if sz := unsafe.Sizeof(hashEntry{}); sz != 128 {
		t.Error("hash entry size mismatch:", sz)
	}
}

func TestHashSet_AddWithLongName(t *testing.T) {
	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()
	zone := InitMetricsZone("TestHashSet_AddWithLongName", 10*1024)
	defer func() {
		zone.Detach()
		Reset()
	}()

	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	length := 200
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	defaultZone.alloc(string(b))
}

func TestHashSet_AddWithMaxLengthName(t *testing.T) {
	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()
	zone := InitMetricsZone("TestHashSet_AddWithMaxLengthName", 10*1024)
	defer func() {
		zone.Detach()
		Reset()
	}()

	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	length := maxNameLength
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	defaultZone.alloc(string(b))
}

func BenchmarkHashSet_Free(b *testing.B) {
	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()
	zone := InitMetricsZone("TestNewSharedMetrics", 30*1024*1024)
	defer func() {
		zone.Detach()
		Reset()
	}()

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
