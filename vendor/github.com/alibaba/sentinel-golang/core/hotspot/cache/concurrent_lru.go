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

package cache

import (
	"sync"
)

// LruCacheMap use LRU strategy to cache the most frequently accessed hotspot parameter
type LruCacheMap struct {
	// Not thread safe
	lru  *LRU
	lock *sync.RWMutex
}

func (c *LruCacheMap) Add(key interface{}, value *int64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.lru.Add(key, value)
	return
}

func (c *LruCacheMap) AddIfAbsent(key interface{}, value *int64) (priorValue *int64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	val := c.lru.AddIfAbsent(key, value)
	if val == nil {
		return nil
	}
	priorValue = val.(*int64)
	return
}

func (c *LruCacheMap) Get(key interface{}) (value *int64, isFound bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	val, found := c.lru.Get(key)
	if found {
		return val.(*int64), true
	}
	return nil, false
}

func (c *LruCacheMap) Remove(key interface{}) (isFound bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.lru.Remove(key)
}

func (c *LruCacheMap) Contains(key interface{}) (ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.lru.Contains(key)
}

func (c *LruCacheMap) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.lru.Keys()
}

func (c *LruCacheMap) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.lru.Len()
}

func (c *LruCacheMap) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.lru.Purge()
}

func NewLRUCacheMap(size int) ConcurrentCounterCache {
	lru, err := NewLRU(size, nil)
	if err != nil {
		return nil
	}
	return &LruCacheMap{
		lru:  lru,
		lock: new(sync.RWMutex),
	}
}
