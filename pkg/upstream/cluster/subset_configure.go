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

package cluster

import (
	"sync"

	"go.uber.org/atomic"
)

func init() {
	UpdateSubsetKeyThresholdConfig(DefaultSubsetKeyThresholdConfig)
}

// SubsetKeyThresholdConfig represents a configuration for logging when key watermarks are exceeded.
// It supports global thresholds, specific key thresholds, and logging control.
type SubsetKeyThresholdConfig struct {
	// EnableLogging: A flag to enable or disable logging.
	EnableLogging bool `json:"enable_logging"`

	// GlobalThreshold: The threshold for any key in the subset.
	// If a key's count exceeds this value, logging is triggered.
	GlobalThreshold int `json:"global_threshold"`

	// SpecificThresholds: A map of specific key thresholds.
	// If a key's count exceeds its corresponding threshold, logging is triggered.
	SpecificThresholds map[string]int `json:"specific_thresholds"`
}

func (s SubsetKeyThresholdConfig) IsEnableLogging() bool {
	return s.EnableLogging
}

func (s SubsetKeyThresholdConfig) GetThresholds(key string) int {
	if value, ok := s.SpecificThresholds[key]; ok {
		return value
	} else {
		return s.GlobalThreshold
	}
}

var DefaultSubsetKeyThresholdConfig = SubsetKeyThresholdConfig{
	EnableLogging:      false,
	GlobalThreshold:    0,
	SpecificThresholds: make(map[string]int, 0),
}

var subsetKeyThresholdConfig atomic.Value

func UpdateSubsetKeyThresholdConfig(config SubsetKeyThresholdConfig) {
	subsetKeyThresholdConfig.Store(config)
}

func LoadSubsetKeyThresholdConfig() SubsetKeyThresholdConfig {
	return subsetKeyThresholdConfig.Load().(SubsetKeyThresholdConfig)
}

// SubsetKeyThresholdListener is an interface that defines a listener for monitoring subset key thresholds.
type SubsetKeyThresholdListener interface {
	// Notify is called when the key's count exceeds the threshold.
	Notify(clusterName string, key string, count int, threshold int)
}

var subsetKeyThresholdListenerManager = &SubsetKeyThresholdListenerManager{}

func LoadSubsetKeyThresholdListenerManager() *SubsetKeyThresholdListenerManager {
	return subsetKeyThresholdListenerManager
}

type SubsetKeyThresholdListenerManager struct {
	writeLock sync.Mutex
	listeners atomic.Value
}

func (sm *SubsetKeyThresholdListenerManager) RegisterSubsetKeyThresholdListener(ln SubsetKeyThresholdListener) {
	sm.writeLock.Lock()
	defer sm.writeLock.Unlock()
	value := sm.listeners.Load()
	if value == nil {
		value = make([]SubsetKeyThresholdListener, 0)
	}
	listeners := value.([]SubsetKeyThresholdListener)
	listeners = append(listeners, ln)
	sm.listeners.Store(listeners)
}

func (sm *SubsetKeyThresholdListenerManager) Notify(clusterName string, key string, count int, threshold int) {
	value := sm.listeners.Load()
	if value == nil {
		return
	}
	listeners := value.([]SubsetKeyThresholdListener)
	for _, listener := range listeners {
		listener.Notify(clusterName, key, count, threshold)
	}
}
