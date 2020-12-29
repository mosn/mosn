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

package network

import (
	"sync"
	"time"
)

// similar to https://github.com/nats-io/nats.go/blob/master/timer.go
var timerPool sync.Pool

// acquireTimer acquires a timer for duration d
func acquireTimer(d time.Duration) *time.Timer {
	if v, ok := timerPool.Get().(*time.Timer); ok && !v.Reset(d) {
		return v
	}

	return time.NewTimer(d)
}

func releaseTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	timerPool.Put(t)
}
