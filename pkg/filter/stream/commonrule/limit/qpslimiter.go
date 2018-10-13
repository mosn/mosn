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
	"sync"
	"github.com/uber/jaeger-lib/metrics"
	"errors"
	"time"
)

type QpsLimiter struct {
	maxAllows int64
	periodMicros int64
	nextPeriodMicros int64
	currentPermits int64

	stopwatch metrics.Stopwatch
	mutex sync.Mutex
}

func NewQpsLimiter(maxAllows int64, periodMs int64) (*QpsLimiter, error) {
	if maxAllows < 0 || periodMs <= 0 {
		return nil, errors.New("maxAllows must not be negtive, and periodMs be positive")
	}

	l :=  &QpsLimiter{
		maxAllows: maxAllows,
		periodMicros: periodMs * int64(time.Millisecond),
		stopwatch: metrics.StartStopwatch(metrics.NullTimer),
	}
	l.nextPeriodMicros = int64(l.stopwatch.ElapsedTime()) + l.periodMicros
	return l, nil
}

func (l *QpsLimiter) TryAcquire() bool {
	if l.maxAllows <= 0 {
		return false;
	}

	l.mutex.Lock()
	var nowMicros = int64(l.stopwatch.ElapsedTime())
	if nowMicros >= l.nextPeriodMicros {
		l.nextPeriodMicros = ((nowMicros - l.nextPeriodMicros) / l.periodMicros + 1) * l.periodMicros + l.nextPeriodMicros;
		l.currentPermits = 0;
	}
	l.currentPermits ++
	defer l.mutex.Unlock()
	return l.currentPermits <= l.maxAllows
}