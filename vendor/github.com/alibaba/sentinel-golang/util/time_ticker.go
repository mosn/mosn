// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"sync/atomic"
	"time"
)

var nowInMs = uint64(0)

// StartTimeTicker starts a background task that caches current timestamp per millisecond,
// which may provide better performance in high-concurrency scenarios.
func StartTimeTicker() {
	atomic.StoreUint64(&nowInMs, uint64(time.Now().UnixNano())/UnixTimeUnitOffset)
	go func() {
		for {
			now := uint64(time.Now().UnixNano()) / UnixTimeUnitOffset
			atomic.StoreUint64(&nowInMs, now)
			time.Sleep(time.Millisecond)
		}
	}()
}

func CurrentTimeMillsWithTicker() uint64 {
	return atomic.LoadUint64(&nowInMs)
}
