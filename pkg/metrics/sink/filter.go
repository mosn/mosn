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

package sink

import "sync"

type FlushFilter struct {
	mutex  sync.RWMutex
	Labels []string
	Keys   []string
}

var flushFilterInstance FlushFilter

// deep copy
func SetFilterLabels(labels []string) {
	flushFilterInstance.mutex.Lock()
	defer flushFilterInstance.mutex.Unlock()
	flushFilterInstance.Labels = make([]string, len(labels))
	copy(flushFilterInstance.Labels, labels)
}

func SetFilterKeys(keys []string) {
	flushFilterInstance.mutex.Lock()
	defer flushFilterInstance.mutex.Unlock()
	flushFilterInstance.Keys = make([]string, len(keys))
	copy(flushFilterInstance.Keys, keys)
}

// If one of the input labels matched the filter's labels, returns true
func IsExclusionLabels(labels []string) bool {
	flushFilterInstance.mutex.RLock()
	defer flushFilterInstance.mutex.RUnlock()
	for _, lb := range flushFilterInstance.Labels {
		for _, l := range labels {
			if l == lb {
				return true
			}
		}
	}
	return false
}

func IsExclusionKeys(key string) bool {
	flushFilterInstance.mutex.RLock()
	defer flushFilterInstance.mutex.RUnlock()
	for _, k := range flushFilterInstance.Keys {
		if k == key {
			return true
		}
	}
	return false
}
