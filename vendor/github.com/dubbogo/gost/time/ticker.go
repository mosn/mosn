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

// Package gxtime encapsulates some golang.time functions
package gxtime

import (
	"time"
)

// Ticker is a wrapper of TimerWheel in golang Ticker style
type Ticker struct {
	C  <-chan time.Time
	ID TimerID
	w  *TimerWheel
}

// NewTicker returns a new Ticker
func NewTicker(d time.Duration) *Ticker {
	if d <= 0 {
		return nil
	}

	return defaultTimerWheel.NewTicker(d)
}

// TickFunc returns a Ticker
func TickFunc(d time.Duration, f func()) *Ticker {
	if d <= 0 {
		return nil
	}

	return defaultTimerWheel.TickFunc(d, f)
}

// Tick is a convenience wrapper for NewTicker providing access to the ticking
// channel only. While Tick is useful for clients that have no need to shut down
// the Ticker, be aware that without a way to shut it down the underlying
// Ticker cannot be recovered by the garbage collector; it "leaks".
// Unlike NewTicker, Tick will return nil if d <= 0.
func Tick(d time.Duration) <-chan time.Time {
	if d <= 0 {
		return nil
	}

	return defaultTimerWheel.Tick(d)
}

// Stop turns off a ticker. After Stop, no more ticks will be sent.
// Stop does not close the channel, to prevent a concurrent goroutine
// reading from the channel from seeing an erroneous "tick".
func (t *Ticker) Stop() {
	(*Timer)(t).Stop()
}

// Reset stops a ticker and resets its period to the specified duration.
// The next tick will arrive after the new period elapses.
func (t *Ticker) Reset(d time.Duration) {
	if d <= 0 {
		return
	}

	(*Timer)(t).Reset(d)
}
