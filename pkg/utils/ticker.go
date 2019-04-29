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
	"runtime/debug"
	"sync/atomic"
	"time"
)

// Ticker is a thread-safe reusable ticker
type Ticker struct {
	innerTicker *time.Ticker
	interval    time.Duration
	callback    func()
	stopChan    chan bool
	started     int32
	stopped     int32
}

func NewTicker(callback func()) *Ticker {
	return &Ticker{
		callback: callback,
		stopChan: make(chan bool, 1),
	}
}

// Start starts a ticker running if it is not started
func (t *Ticker) Start(interval time.Duration) {
	if !atomic.CompareAndSwapInt32(&t.started, 0, 1) {
		return
	}

	if t.innerTicker == nil {
		t.innerTicker = time.NewTicker(interval)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				debug.PrintStack()
			}
			t.Close()
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

// Stop stops the ticker.
func (t *Ticker) Stop() {
	if !atomic.CompareAndSwapInt32(&t.stopped, 0, 1) {
		return
	}

	t.stopChan <- true
}

// Close closes the ticker.
func (t *Ticker) Close() {
	close(t.stopChan)
}
