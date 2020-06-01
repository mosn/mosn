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

package regulator

import (
	"fmt"
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/util"
)

const MAX_NO_INVOKE_CYCLE = 6

type InvocationStat struct {
	dimension      InvocationDimension
	host           api.HostInfo
	callCount      uint64
	exceptionCount uint64
	uselessCycle   int32
	downgradeTime  int64
}

func NewInvocationStat(host api.HostInfo, dimension InvocationDimension) *InvocationStat {
	invocationStat := &InvocationStat{
		dimension:      dimension,
		host:           host,
		callCount:      0,
		exceptionCount: 0,
		uselessCycle:   0,
		downgradeTime:  0,
	}
	return invocationStat
}

func (s *InvocationStat) Call(isException bool) {
	atomic.AddUint64(&s.callCount, 1)
	if isException {
		atomic.AddUint64(&s.exceptionCount, 1)
	}
}

func (s *InvocationStat) GetMeasureKey() string {
	return s.dimension.GetMeasureKey()
}

func (s *InvocationStat) GetInvocationKey() string {
	return s.dimension.GetInvocationKey()
}

func (s *InvocationStat) GetCall() uint64 {
	return atomic.LoadUint64(&s.callCount)
}

func (s *InvocationStat) GetCount() (uint64, uint64) {
	return atomic.LoadUint64(&s.callCount), atomic.LoadUint64(&s.exceptionCount)
}

func (s *InvocationStat) GetDowngradeTime() int64 {
	return s.downgradeTime
}

func (s *InvocationStat) AddUselessCycle() bool {
	s.uselessCycle++
	return s.uselessCycle >= MAX_NO_INVOKE_CYCLE
}

func (s *InvocationStat) RestUselessCycle() {
	s.uselessCycle = 0
}

func (s *InvocationStat) GetExceptionRate() (bool, float64) {
	if atomic.LoadUint64(&s.callCount) <= 0 {
		return false, 0
	}
	return true, util.DivideInt64(int64(atomic.LoadUint64(&s.exceptionCount)), int64(atomic.LoadUint64(&s.callCount)))
}

func (s *InvocationStat) Snapshot() *InvocationStat {
	return &InvocationStat{
		dimension:      s.dimension,
		host:           s.host,
		callCount:      atomic.LoadUint64(&s.callCount),
		exceptionCount: s.exceptionCount,
		uselessCycle:   s.uselessCycle,
		downgradeTime:  s.downgradeTime,
	}
}

func (s *InvocationStat) Update(snapshot *InvocationStat) {
	call, exception := snapshot.GetCount()
	// To subtract a signed positive constant value c from x, do AddUint64(&x, ^uint64(c-1)).
	atomic.AddUint64(&s.exceptionCount, ^uint64(exception-1))
	atomic.AddUint64(&s.callCount, ^uint64(call-1))
}

func (s *InvocationStat) Downgrade() {
	(s.host).SetHealthFlag(api.FAILED_OUTLIER_CHECK)
	s.downgradeTime = util.GetNowMS()
}

func (s *InvocationStat) Recover() {
	atomic.StoreUint64(&s.exceptionCount, 0)
	atomic.StoreUint64(&s.callCount, 0)
	s.uselessCycle = 0
	s.downgradeTime = 0
	if s.host != nil {
		(s.host).ClearHealthFlag(api.FAILED_OUTLIER_CHECK)
	}
}

func (s *InvocationStat) IsHealthy() bool {
	return (s.host).Health()
}

func (s *InvocationStat) String() string {
	str := fmt.Sprintf("host=%s, dimension=%s,callCount=%v,exceptionCount=%v,uselessCycle=%v, downgradeTime=%v",
		(s.host).AddressString(), s.dimension, atomic.LoadUint64(&s.callCount), atomic.LoadUint64(&s.exceptionCount), s.uselessCycle, s.downgradeTime)
	return str
}
