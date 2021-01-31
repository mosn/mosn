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

// ConcurrentCounterCache cache the hotspot parameter
type ConcurrentCounterCache interface {
	// Add add a value to the cache,
	// Updates the "recently used"-ness of the key.
	Add(key interface{}, value *int64)

	// If the key is not existed in the cache, adds a value to the cache then return nil. And updates the "recently used"-ness of the key
	// If the key is already existed in the cache, do nothing and return the prior value
	AddIfAbsent(key interface{}, value *int64) (priorValue *int64)

	// Get returns key's value from the cache and updates the "recently used"-ness of the key.
	Get(key interface{}) (value *int64, isFound bool)

	// Remove removes a key from the cache.
	// Return true if the key was contained.
	Remove(key interface{}) (isFound bool)

	// Contains checks if a key exists in cache
	// Without updating the recent-ness.
	Contains(key interface{}) (ok bool)

	// Keys returns a slice of the keys in the cache, from oldest to newest.
	Keys() []interface{}

	// Len returns the number of items in the cache.
	Len() int

	// Purge clears all cache entries.
	Purge()
}
