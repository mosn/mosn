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

	"github.com/stretchr/testify/assert"

	"mosn.io/mosn/pkg/types"
)

func TestCounterFunc(t *testing.T) {
	f := NewShmCounterFunc("mosn_test")
	m := f()
	m.Inc(111)
	assert.Equal(t, m.Count(), int64(111))
}

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
	assert.Nil(t, err)

	// inc
	counter := ShmCounter(unsafe.Pointer(&entry.value))
	defer counter.Stop()
	counter.Inc(5)

	assert.Equal(t, counter.Count(), int64(5))

	// dec
	counter.Dec(2)
	assert.Equal(t, counter.Count(), int64(3))

	cs := counter.Snapshot()
	assert.Equal(t, cs.Count(), int64(3))

	// clear
	counter.Clear()
	assert.Equal(t, counter.Count(), int64(0))
	if counter.Count() != 0 {
		t.Error("count ops failed")
	}

}
