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
	"sync"
	"sync/atomic"
	"time"
)

const (
	updateInit = iota
	updating
	updateFailed
)

const NeverExpire time.Duration = -1

type expiredata struct {
	data        interface{}
	expiredTime time.Time
	valid       time.Duration
	updated     uint32
}

func (d *expiredata) checkValid() bool {
	// if valid is equal NeverExpire and don't need to update.
	if d.valid == NeverExpire {
		return true
	}

	return d.expiredTime.After(time.Now())
}

type ExpiredMap struct {
	syncMap *sync.Map
	syncMod bool // true means using synchronous update mode, otherwise async mod

	// When the cache expires, it is used to update the cache.
	UpdateHandler func(interface{}) (interface{}, bool)
}

// handler is used to update the data if the cache is invalid during Get.
// syncMod is set true means that the handler is called synchronously, and the others are asynchronous.
func NewExpiredMap(handler func(interface{}) (interface{}, bool), syncMod bool) *ExpiredMap {
	return &ExpiredMap{
		syncMap:       &sync.Map{},
		UpdateHandler: handler,
		syncMod:       syncMod,
	}
}

// Set a key and val with an expiration time.
// key and val represent cached index and user data.
// valid is used to set the expire time of the cache. For example, if valid=10 means the data expires after 10 Duration.
func (e *ExpiredMap) Set(key, val interface{}, valid time.Duration) {
	ct := time.Now()
	e.syncMap.Store(key, &expiredata{data: val, expiredTime: ct.Add(valid), valid: valid})
}

// Get the cache indexed by key.
// If the cache is hit, the bool value indicates whether the cache is expired.
func (e *ExpiredMap) Get(key interface{}) (interface{}, bool) {
	if val, ok := e.syncMap.Load(key); ok {
		eval := val.(*expiredata)
		if ok := eval.checkValid(); ok {
			// if updated success
			if atomic.LoadUint32(&eval.updated) == 0 {
				return eval.data, true
			} else {
				return eval.data, false
			}

		}

		// Cache expires, updated via updateHandler.
		// Check eval.updated to avoid cache flood.
		if e.UpdateHandler != nil && (atomic.CompareAndSwapUint32(&eval.updated, updateInit, 1) || atomic.CompareAndSwapUint32(&eval.updated, updateFailed, 1)) {

			e.updateData(key, eval.valid)
			// If it is a synchronous update mode, get data again.
			if e.syncMod {
				if val, ok := e.syncMap.Load(key); ok {
					eval := val.(*expiredata)
					if ok := eval.checkValid(); ok {
						if atomic.LoadUint32(&eval.updated) == 0 {
							return eval.data, true
						} else {
							return eval.data, false
						}

					}
				}
			}
		}

		return eval.data, false
	}

	// When the cache is not hit, not update actively update.
	return nil, false
}

func (e *ExpiredMap) updateData(key interface{}, valid time.Duration) {
	updater := func() {
		if newVal, ok := e.UpdateHandler(key); ok {
			ct := time.Now()
			e.syncMap.Store(key, &expiredata{data: newVal, expiredTime: ct.Add(valid), valid: valid})
			return
		}

		// Set expires time is half of 'valid' when update handler failed
		if val, ok := e.syncMap.Load(key); ok {
			eval := val.(*expiredata)
			ct := time.Now()
			e.syncMap.Store(key, &expiredata{data: eval.data, expiredTime: ct.Add(valid / 2), valid: valid, updated: updateFailed})
		}
	}

	if e.syncMod {
		updater()

	} else {
		GoWithRecover(updater, nil)
	}

}
