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
	"math/rand"
	"testing"
	"time"

	"github.com/cch123/supermonkey"
	"github.com/stretchr/testify/assert"
)

const delta = 1e-6

func TestEWMA_decay(t *testing.T) {
	var now time.Time
	supermonkey.Patch(time.Now, func() time.Time {
		return now
	})

	var startTime time.Time
	tests := []struct {
		duration      time.Duration
		exceptedAlpha float64
		exceptedRate  float64
	}{
		{duration: 0, exceptedAlpha: 0.9932620530009145, exceptedRate: 0.006692547069322991},
		{duration: 1 * time.Second, exceptedAlpha: 0.9932620530009145, exceptedRate: 0.006692547069322991},
		{duration: 5 * time.Second, exceptedAlpha: 0.6321205588285577, exceptedRate: 0.004259194822419109},
		{duration: 15 * time.Second, exceptedAlpha: 0.28346868942621073, exceptedRate: 0.001909997005254027},
		{duration: 1 * time.Minute, exceptedAlpha: 0.07995558537067671, exceptedRate: 0.0005387364965084747},
		{duration: 5 * time.Minute, exceptedAlpha: 0.01652854617838251, exceptedRate: 0.00011136846812187828},
		{duration: 15 * time.Minute, exceptedAlpha: 0.005540151995103271, exceptedRate: 3.7329250509883136e-05},
	}

	for _, tt := range tests {
		now = startTime
		alpha := Alpha(math.Exp(-5), tt.duration)
		assert.InDelta(t, tt.exceptedAlpha, alpha, delta)

		ewma := NewEWMA(alpha)
		ewma.Update(1)
		now = now.Add(time.Second)
		assert.InDelta(t, alpha, ewma.Rate(), delta)

		if tt.duration == 0 {
			now = now.Add(minDecayDuration)
		} else {
			now = now.Add(tt.duration)
		}

		assert.InDelta(t, tt.exceptedRate, ewma.Rate(), delta)
	}

}

func TestEWMA_reduceTick(t *testing.T) {
	var now time.Time
	supermonkey.Patch(time.Now, func() time.Time {
		return now
	})

	now = time.Now()

	alpha := Alpha(math.Exp(-5), time.Second)
	ewma := NewEWMA(alpha)

	for i := 0; i < 100; i++ {
		ewma.Update(1)
	}

	now = now.Add(time.Second)
	assert.InDelta(t, 0.9932620530009145, ewma.Rate(), delta)
}

func TestEWMA_uncounted(t *testing.T) {
	var now time.Time
	supermonkey.Patch(time.Now, func() time.Time {
		return now
	})

	var startTime time.Time
	tests := []struct {
		duration     time.Duration
		exceptedRate float64
	}{
		{duration: 0, exceptedRate: 0.9933071470283565},
		{duration: 1 * time.Second, exceptedRate: 0.9933071470283565},
		{duration: 5 * time.Second, exceptedRate: 0.6336874290396695},
		{duration: 15 * time.Second, exceptedRate: 0.2848372620835774},
		{duration: 1 * time.Minute, exceptedRate: 0.0804512468752463},
		{duration: 5 * time.Minute, exceptedRate: 0.016638073887636218},
		{duration: 15 * time.Minute, exceptedRate: 0.005577274435891465},
	}

	for _, tt := range tests {
		now = startTime
		alpha := Alpha(math.Exp(-5), tt.duration)

		ewma := NewEWMA(alpha)
		ewma.Update(1)
		now = now.Add(time.Second)
		assert.InDelta(t, alpha, ewma.Rate(), delta)
		// flushed but still previous belongs to the interval
		assert.InDelta(t, alpha, ewma.Rate(), delta)

		if tt.duration == 0 {
			now = now.Add(minDecayDuration)
		} else {
			now = now.Add(tt.duration)
		}

		ewma.Tick()

		for i := 0; i < rand.Intn(10)+1; i++ {
			ewma.Update(1)
		}

		assert.InDelta(t, tt.exceptedRate, ewma.Rate(), delta)
		assert.InDelta(t, tt.exceptedRate, ewma.Snapshot().Rate(), delta)
	}

	now = startTime
	alpha := Alpha(math.Exp(-5), time.Second)
	ewma := NewEWMA(alpha)
	now = now.Add(time.Nanosecond)
	assert.InDelta(t, float64(0), ewma.Rate(), delta)
}

func TestAlpha(t *testing.T) {
	tests := []time.Duration{
		1 * time.Second,
		5 * time.Second,
		15 * time.Second,
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
	}

	for _, tt := range tests {
		alpha := Alpha(0.001, tt)
		assert.InDelta(t, 0.001, math.Pow(1-alpha, float64(tt/time.Second)), delta)
	}
}
