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

package cluster

import (
	"sync"
	"sync/atomic"

	"mosn.io/api"
)

// health flag resue for same address
// TODO: use one map for all reuse data
var healthStore = sync.Map{}

func GetHealthFlagPointer(addr string) *uint64 {
	v, _ := healthStore.LoadOrStore(addr, func() *uint64 {
		f := uint64(0)
		return &f
	}())
	p, _ := v.(*uint64)
	return p
}

func SetHealthFlag(p *uint64, flag api.HealthFlag) {
	if p == nil {
		return
	}
	f := atomic.LoadUint64(p)
	f |= uint64(flag)
	atomic.StoreUint64(p, f)
}

func ClearHealthFlag(p *uint64, flag api.HealthFlag) {
	if p == nil {
		return
	}
	f := atomic.LoadUint64(p)
	f &= ^uint64(flag)
	atomic.StoreUint64(p, f)
}
