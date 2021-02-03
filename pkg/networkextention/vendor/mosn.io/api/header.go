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

package api

// HeaderMap is a interface to provide operation facade with user-value headers
type HeaderMap interface {
	// Get value of key
	// If multiple values associated with this key, first one will be returned.
	Get(key string) (string, bool)

	// Set key-value pair in header map, the previous pair will be replaced if exists
	Set(key, value string)

	// Add value for given key.
	// Multiple headers with the same key may be added with this function.
	// Use Set for setting a single header for the given key.
	Add(key, value string)

	// Del delete pair of specified key
	Del(key string)

	// Range calls f sequentially for each key and value present in the map.
	// If f returns false, range stops the iteration.
	Range(f func(key, value string) bool)

	// Clone used to deep copy header's map
	Clone() HeaderMap

	// ByteSize return size of HeaderMap
	ByteSize() uint64
}
