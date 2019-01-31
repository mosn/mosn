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

package utils

import (
	"sync/atomic"
	"time"
)

// thread-safe reusable timer
type Timer struct {
	callback   func()
	interval   time.Duration
	innerTimer *time.Timer
	stopped    int32
	started    int32
	stopChan   chan bool
}

func NewTimer(callback func()) *Timer {
	return &Timer{
		callback: callback,
		stopChan: make(chan bool, 1),
	}
}

// Start starts a timer if it is not started
func (t *Timer) Start(interval time.Duration) {
	if !atomic.CompareAndSwapInt32(&t.started, 0, 1) {
		return
	}

	if t.innerTimer == nil {
		t.innerTimer = time.NewTimer(interval)
	} else {
		t.innerTimer.Reset(interval)
	}

	go func() {
		defer func() {
			t.innerTimer.Stop()
			atomic.StoreInt32(&t.started, 0)
			atomic.StoreInt32(&t.stopped, 0)
		}()

		select {
		case <-t.innerTimer.C:
			t.callback()
		case <-t.stopChan:
			return
		}
	}()

}

// Stop stops the timer.
func (t *Timer) Stop() {
	if !atomic.CompareAndSwapInt32(&t.stopped, 0, 1) {
		return
	}

	t.stopChan <- true
}

// Close closes the timers.
func (t *Timer) Close() {
	close(t.stopChan)
}
