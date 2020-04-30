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
	"testing"
	"unsafe"

	"mosn.io/mosn/pkg/types"
)

func TestCounter(t *testing.T) {
	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()
	zone := InitMetricsZone("TestCounter", 10*1024)
	defer func() {
		zone.Detach()
		Reset()
	}()

	entry, err := defaultZone.alloc("TestCounter")
	if err != nil {
		t.Fatal(err)
	}
	// inc
	counter := ShmCounter(unsafe.Pointer(&entry.value))
	counter.Inc(5)

	if counter.Count() != 5 {
		t.Error("count ops failed")
	}

	// dec
	counter.Dec(2)
	if counter.Count() != 3 {
		t.Error("count ops failed")
	}

	// clear
	counter.Clear()
	if counter.Count() != 0 {
		t.Error("count ops failed")
	}
}
