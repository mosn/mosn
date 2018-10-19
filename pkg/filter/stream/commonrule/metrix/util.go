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

package metrix

import (
	"sync/atomic"
	"time"
)

// thread-safe reusable timer
type timer struct {
	callback   func()
	interval   time.Duration
	innerTimer *time.Timer
	stopped    int32
	started    int32
	stopChan   chan bool
}

func newTimer(callback func()) *timer {
	return &timer{
		callback: callback,
		stopChan: make(chan bool, 1),
	}
}

func (t *timer) start(interval time.Duration) {
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

func (t *timer) stop() {
	if !atomic.CompareAndSwapInt32(&t.stopped, 0, 1) {
		return
	}

	t.stopChan <- true
}

func (t *timer) close() {
	close(t.stopChan)
}

// thread-safe reusable ticker
type ticker struct {
	innerTicker *time.Ticker
	interval    time.Duration
	callback    func()
	stopChan    chan bool
	started     int32
	stopped     int32
}

// NewTicker new
func NewTicker(callback func()) *ticker {
	return &ticker{
		callback: callback,
		stopChan: make(chan bool, 1),
	}
}

// Start start
func (t *ticker) Start(interval time.Duration) {
	if !atomic.CompareAndSwapInt32(&t.started, 0, 1) {
		return
	}

	if t.innerTicker == nil {
		t.innerTicker = time.NewTicker(interval)
	}

	go func() {
		defer func() {
			// close chan when health check stopped
			t.close()
			atomic.StoreInt32(&t.started, 0)
			atomic.StoreInt32(&t.stopped, 0)
		}()

		for {
			select {
			case <-t.innerTicker.C:
				t.callback()
			case <-t.stopChan:
				t.innerTicker.Stop()
				return
			}
		}
	}()
}

func (t *ticker) Stop() {
	if !atomic.CompareAndSwapInt32(&t.stopped, 0, 1) {
		return
	}

	t.stopChan <- true
}

func (t *ticker) close() {
	close(t.stopChan)
}
