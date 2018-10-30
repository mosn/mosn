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

package limit

import (
	"errors"
	"math"
	"sync"
	"time"
)

// RateLimiter limit
type RateLimiter struct {
	maxAllows            int64
	maxPermits           float64
	stableIntervalMicros float64
	storedPermits        float64
	nextFreeTicketMicros int64

	start time.Time
	mutex sync.Mutex
}

// NewRateLimiter new
func NewRateLimiter(maxAllows int64, periodMs int64, MaxBurstRatio float64) (*RateLimiter, error) {
	if maxAllows < 0 || periodMs <= 0 || MaxBurstRatio <= 0 {
		return nil, errors.New("maxAllows must not be negtive, and periodMs be positive, and maxBurstTimes be positive")
	}
	var interval float64
	if 0 == maxAllows {
		interval = float64(time.Millisecond)
	} else {
		interval = float64(periodMs) * float64(time.Millisecond) / float64(maxAllows)
	}
	l := &RateLimiter{
		maxAllows:            maxAllows,
		maxPermits:           MaxBurstRatio * float64(maxAllows),
		stableIntervalMicros: interval,
		start:                time.Now(),
	}
	l.nextFreeTicketMicros = int64(time.Since(l.start))

	return l, nil
}

// TryAcquire limit
func (l *RateLimiter) TryAcquire() bool {
	if l.maxAllows <= 0 {
		return false
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	nowMicros := int64(time.Since(l.start))
	if nowMicros <= l.nextFreeTicketMicros {
		return false
	}
	l.reserveEarliestAvailable(nowMicros)
	return true
}

// calculate nextFreeTicket time and storedPermits
func (l *RateLimiter) reserveEarliestAvailable(nowMicros int64) {
	//calculate new permits and update storedPermits
	newPermits := float64(nowMicros-l.nextFreeTicketMicros) / l.stableIntervalMicros
	l.storedPermits = math.Min(l.maxPermits, l.storedPermits+newPermits)
	l.nextFreeTicketMicros = nowMicros

	//calculate next free ticket timestamp
	storedPermitsToSpend := math.Min(1, l.storedPermits)
	freshPermits := 1 - storedPermitsToSpend
	waitMicros := int64(freshPermits * l.stableIntervalMicros)

	l.nextFreeTicketMicros = l.nextFreeTicketMicros + waitMicros
	l.storedPermits -= storedPermitsToSpend
}
