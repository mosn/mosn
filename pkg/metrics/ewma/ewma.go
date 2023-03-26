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

package ewma

import (
	"math"
	"sync"
	"time"

	gometrics "github.com/rcrowley/go-metrics"
)

var nowFunc = time.Now

const (
	minDecayDuration = time.Second
)

// EWMA is the two level implementation of an EWMA.
// The first level will be counted according to the second time interval to get the arithmetic mean,
// and then decay it through the exponential moving weighted average (EWMA).
// See: https://en.wikipedia.org/wiki/Moving_average#Exponential_moving_average
type EWMA struct {
	alpha float64

	uncountedSum   int64
	uncountedCount int64

	lastEWMA     float64
	lastTickTime time.Time
	mutex        sync.Mutex
}

func NewEWMA(alpha float64) gometrics.EWMA {
	return &EWMA{
		alpha:        alpha,
		lastTickTime: nowFunc(),
	}
}

func (e *EWMA) Rate() float64 {
	e.flush(nowFunc())
	return e.lastEWMA
}

func (e *EWMA) Snapshot() gometrics.EWMA {
	return gometrics.EWMASnapshot(e.Rate())
}

func (e *EWMA) Tick() {
	e.flush(nowFunc())
}

func (e *EWMA) Update(i int64) {
	e.mutex.Lock()
	e.uncountedSum += i
	e.uncountedCount++
	e.mutex.Unlock()
}

func (e *EWMA) flush(now time.Time) {
	e.mutex.Lock()
	duration := now.Sub(e.lastTickTime)

	if duration >= time.Second {
		sum := e.uncountedSum
		count := e.uncountedCount

		if count > 0 {
			e.uncountedSum = 0
			e.uncountedCount = 0
			e.lastEWMA = e.ewma(float64((sum+count-1)/count), now)
		} else {
			e.lastEWMA = e.ewma(0, now)
		}

		e.lastTickTime = now
	}
	e.mutex.Unlock()
}

func (e *EWMA) ewma(i float64, now time.Time) float64 {
	return i*e.alpha + math.Pow(1-e.alpha, float64(now.Sub(e.lastTickTime)/time.Second))*e.lastEWMA
}

// Alpha the alpha needed to decay 1 to negligible (less than target) over a given duration.
//
// (1 - alpha) ^ duration = target     ==>     alpha = 1 - target ^ (1 / duration)
func Alpha(target float64, duration time.Duration) float64 {
	if duration < minDecayDuration {
		duration = minDecayDuration
	}
	return 1 - math.Pow(target, 1/float64(duration/time.Second))
}
