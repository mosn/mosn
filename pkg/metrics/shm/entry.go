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
)

const maxNameLength = 107

// metricsEntry is the mapping for metrics entry record memory-layout in shared memory.
//
// This struct should never be instantiated.
type metricsEntry struct {
	value int64                   // 8
	ref   uint32                  // 4
	name  [maxNameLength + 1]byte // 107 + 1 for '\0', end of string tag in C-style string
}

func (e *metricsEntry) assignName(name []byte) {
	i := 0
	for i = range name {
		e.name[i] = name[i]
	}
	e.name[i+1] = 0
}

func (e *metricsEntry) equalName(name []byte) bool {
	i := 0
	for i = range name {
		if e.name[i] != name[i] {
			return false
		}
	}
	// no more characters
	return e.name[i+1] == 0
}

func (e *metricsEntry) getName() []byte {
	for i := 0; i < len(e.name); i++ {
		if e.name[i] == 0 {
			return e.name[:i]
		}
	}
	return e.name[:]
}

func (e *metricsEntry) incRef() {
	atomic.AddUint32(&e.ref, 1)
}

func (e *metricsEntry) decRef() bool {
	return atomic.AddUint32(&e.ref, ^uint32(0)) == 0
}
