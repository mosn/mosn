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

// Timer is a wrapper of TimeWheel to supply go timer funcs
type Timer struct {
	C  <-chan time.Time
	ID TimerID
	w  *TimerWheel
}

// After waits for the duration to elapse and then sends the current time
// on the returned channel.
func After(d time.Duration) <-chan time.Time {
	if d <= 0 {
		return nil
	}

	return defaultTimerWheel.After(d)
}

// Sleep pauses the current goroutine for at least the duration d.
// A negative or zero duration causes Sleep to return immediately.
func Sleep(d time.Duration) {
	if d <= 0 {
		return
	}

	defaultTimerWheel.Sleep(d)
}

// AfterFunc waits for the duration to elapse and then calls f
// in its own goroutine. It returns a Timer that can
// be used to cancel the call using its Stop method.
func AfterFunc(d time.Duration, f func()) *Timer {
	if d <= 0 {
		return nil
	}

	return defaultTimerWheel.AfterFunc(d, f)
}

// NewTimer creates a new Timer that will send
// the current time on its channel after at least duration d.
func NewTimer(d time.Duration) *Timer {
	if d <= 0 {
		return nil
	}

	return defaultTimerWheel.NewTimer(d)
}

// Reset changes the timer to expire after duration d.
// It returns true if the timer had been active, false if the timer had
// expired or been stopped.
func (t *Timer) Reset(d time.Duration) {
	if d <= 0 {
		return
	}
	if t.w == nil {
		panic("time: Stop called on uninitialized Timer")
	}

	_ = t.w.resetTimer(t, d)
}

// Stop prevents the Timer from firing.
func (t *Timer) Stop() {
	if t.w == nil {
		panic("time: Stop called on uninitialized Timer")
	}

	_ = t.w.deleteTimer(t)
	t.w = nil
}
