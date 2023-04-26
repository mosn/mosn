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

package faultinject

import (
	"math/rand"
	"sync/atomic"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type faultInjector struct {
	// 1~100
	delayPercent  uint32
	delayDuration uint64
	delaying      uint32
	readCallbacks api.ReadFilterCallbacks
}

// NewFaultInjector makes a fault injector as types.ReadFilter
func NewFaultInjector(config *v2.FaultInject) api.ReadFilter {
	return &faultInjector{
		delayPercent:  config.DelayPercent,
		delayDuration: config.DelayDuration,
	}
}

func (fi *faultInjector) OnData(buffer types.IoBuffer) api.FilterStatus {
	fi.tryInjectDelay()

	if atomic.LoadUint32(&fi.delaying) > 0 {
		return api.Stop
	}

	return api.Continue
}

func (fi *faultInjector) OnNewConnection() api.FilterStatus {
	return api.Continue
}

func (fi *faultInjector) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {
	fi.readCallbacks = cb
}

func (fi *faultInjector) tryInjectDelay() {
	if atomic.LoadUint32(&fi.delaying) > 0 {
		return
	}

	duration := fi.getDelayDuration()

	if duration > 0 {
		if atomic.CompareAndSwapUint32(&fi.delaying, 0, 1) {
			utils.GoWithRecover(func() {
				select {
				case <-time.After(time.Duration(duration)):
					atomic.StoreUint32(&fi.delaying, 0)
					fi.readCallbacks.ContinueReading()
				}
			}, nil)
		}
	}
}

func (fi *faultInjector) getDelayDuration() uint64 {
	if fi.delayPercent == 0 {
		return 0
	}

	if uint32(rand.Intn(100))+1 > fi.delayPercent {
		return 0
	}

	return fi.delayDuration
}
