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

package metrics

type metricsMatcher struct {
	rejectAll       bool
	exclusionLabels []string
	exclusionKeys   []string
}

// isExclusionLabels returns the labels will be ignored or not
func (m *metricsMatcher) isExclusionLabels(labels map[string]string) bool {
	if m.rejectAll {
		return true
	}
	// TODO: support pattern
	for _, label := range m.exclusionLabels {
		if _, ok := labels[label]; ok {
			return true
		}
	}
	return false
}

// isExclusionKey returns the key will be ignored or not
func (m *metricsMatcher) isExclusionKey(key string) bool {
	if m.rejectAll {
		return true
	}
	// TODO: support pattern
	for _, eKey := range m.exclusionKeys {
		if eKey == key {
			return true
		}
	}
	return false
}
