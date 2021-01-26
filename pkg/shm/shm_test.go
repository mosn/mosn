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
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"mosn.io/mosn/pkg/types"
)

func TestAtomic(t *testing.T) {
	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()

	span, err := Alloc("TestAtomic", 256)
	if err != nil {
		t.Error(err)
	}
	block, err := span.Alloc(128)

	counter := (*uint32)(unsafe.Pointer(block))
	expected := 10000
	cpu := runtime.GOMAXPROCS(-1)
	wg := sync.WaitGroup{}

	wg.Add(cpu)
	for i := 0; i < cpu; i++ {
		go func() {
			for j := 0; j < expected; j++ {
				atomic.AddUint32(counter, 1)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	if *counter != uint32(expected*cpu) {
		t.Errorf("counter error, expected %d, actual %d", expected*cpu, *counter)
	}

	if err := Free(span); nil != err {
		log.Fatalln(err)
	}

}

func TestConsitency(t *testing.T) {
	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()

	span, err := Alloc("TestConsitency", 256)
	if err != nil {
		t.Error(err)
	}

	_, err = Alloc("TestConsitency", 512)
	if err == nil && err.Error() == "mmap target path ./mosn_shm_TestConsitency exists and its size 256 mismatch 512" {
		t.Error()
	}

	if span.Data() != uintptr(unsafe.Pointer(&(span.Origin())[0])) {
		t.Error()
	}

	if err := Free(span); nil != err {
		log.Fatalln(err)
	}

}

func BenchmarkPointerCast_Raw(b *testing.B) {
	var counter uint32 = 0
	ptr := &counter
	for i := 0; i < b.N; i++ {
		atomic.AddUint32(ptr, 1)
	}
}

func BenchmarkPointerCast_Cast(b *testing.B) {
	var counter uint32 = 0
	ptr := uintptr(unsafe.Pointer(&counter))
	for i := 0; i < b.N; i++ {
		atomic.AddUint32((*uint32)(unsafe.Pointer(ptr)), 1)
	}
}
