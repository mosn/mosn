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

const (
	minDecayDuration = time.Second
)

// EWMA is the standard EWMA implementation, it updates in real time
// and when queried it always returns the decayed value.
// See: https://en.wikipedia.org/wiki/Moving_average#Exponential_moving_average
type EWMA struct {
	alpha float64

	lastEWMA     float64
	lastTickTime time.Time
	mutex        sync.Mutex
}

// NewEWMA constructs a new EWMA with the given alpha.
func NewEWMA(alpha float64) gometrics.EWMA {
	return &EWMA{
		alpha:        alpha,
		lastTickTime: time.Now(),
	}
}

// Rate returns the moving average mean of events per second.
func (e *EWMA) Rate() float64 {
	return e.ewma(0, time.Now())
}

// Snapshot returns a read-only copy of the EWMA.
func (e *EWMA) Snapshot() gometrics.EWMA {
	return gometrics.EWMASnapshot(e.Rate())
}

// Tick ticks the clock to update the moving average.
// There is no need to use an additional timer to Tick in this implementation,
// because Rate also calculates the latest value when it is updated or queried.
func (e *EWMA) Tick() {
	e.Update(0)
}

// Update adds an uncounted event with value `i`, and tries to flush.
func (e *EWMA) Update(i int64) {
	now := time.Now()

	e.mutex.Lock()
	e.lastEWMA = e.ewma(float64(i), now)
	e.lastTickTime = now
	e.mutex.Unlock()
}

func (e *EWMA) ewma(i float64, now time.Time) float64 {
	return i*e.alpha + math.Pow(1-e.alpha, float64(now.Sub(e.lastTickTime))/float64(time.Second))*e.lastEWMA
}

// Alpha the alpha needed to decay 1 to negligible (less than target) over a given duration.
//
// (1 - alpha) ^ duration = target     ==>     alpha = 1 - target ^ (1 / duration).
func Alpha(target float64, duration time.Duration) float64 {
	if duration < minDecayDuration {
		duration = minDecayDuration
	}

	return 1 - math.Pow(target, 1/float64(duration/time.Second))
}
