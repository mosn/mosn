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
	"strconv"
	"sync/atomic"
	"time"
)

var (
	// lastTime is used to cache time
	lastTime atomic.Value
)

// timeCache is used to reduce format
type timeCache struct {
	t int64
	s string
}

// CacheTime returns a time cache in seconds.
// we use a cache to reduce the format
func CacheTime() string {
	var s string
	t := time.Now()
	nano := t.UnixNano()
	now := nano / 1e9
	value := lastTime.Load()
	if value != nil {
		last := value.(*timeCache)
		if now <= last.t {
			s = last.s
		}
	}
	if s == "" {
		s = t.Format("2006-01-02 15:04:05")
		lastTime.Store(&timeCache{now, s})
	}
	mi := nano % 1e9 / 1e6
	s = s + "," + strconv.Itoa(int(mi))
	return s
}
