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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ SubsetKeyThresholdListener = &MockSubsetKeyThresholdListener{}

func TestSubsetKeyThresholdConfig(t *testing.T) {
	assert := assert.New(t)

	defaultConfig := LoadSubsetKeyThresholdConfig()
	assert.True(reflect.DeepEqual(defaultConfig, DefaultSubsetKeyThresholdConfig))
	assert.False(defaultConfig.IsEnableLogging())
	assert.Equal(0, defaultConfig.GetThresholds("mosn_aig"))

	newConfig := SubsetKeyThresholdConfig{
		EnableLogging:   true,
		GlobalThreshold: 100,
		SpecificThresholds: map[string]int{
			"mosn_aig": 200,
			"zone":     30,
		},
	}

	UpdateSubsetKeyThresholdConfig(newConfig)
	reloadConfig := LoadSubsetKeyThresholdConfig()
	assert.True(reflect.DeepEqual(newConfig, reloadConfig))
	assert.True(reloadConfig.IsEnableLogging())

	assert.Equal(200, reloadConfig.GetThresholds("mosn_aig"))
	assert.Equal(30, reloadConfig.GetThresholds("zone"))
	assert.Equal(100, reloadConfig.GetThresholds("mosn_version"))
}

func TestSubsetKeyThresholdListenerManager(t *testing.T) {
	assert := assert.New(t)

	// Expect no panic
	LoadSubsetKeyThresholdListenerManager().Notify("clusterA", "key", 10, 5)

	listener := NewMockSubsetKeyThresholdListener()
	LoadSubsetKeyThresholdListenerManager().RegisterSubsetKeyThresholdListener(listener)

	LoadSubsetKeyThresholdListenerManager().Notify("clusterA", "key1", 10, 5)
	LoadSubsetKeyThresholdListenerManager().Notify("clusterB", "key2", 11, 6)
	LoadSubsetKeyThresholdListenerManager().Notify("clusterC", "key3", 12, 7)

	events := listener.GetEvents()
	assert.NotNil(events)
	assert.Equal(3, len(events))
	assert.True(MockEvent{ClusterName: "clusterA", Key: "key1", Count: 10, Threshold: 5}.Equal(events[0]))
	assert.True(MockEvent{ClusterName: "clusterB", Key: "key2", Count: 11, Threshold: 6}.Equal(events[1]))
	assert.True(MockEvent{ClusterName: "clusterC", Key: "key3", Count: 12, Threshold: 7}.Equal(events[2]))
}

type MockSubsetKeyThresholdListener struct {
	events []MockEvent
}

func NewMockSubsetKeyThresholdListener() *MockSubsetKeyThresholdListener {
	return &MockSubsetKeyThresholdListener{
		events: make([]MockEvent, 0),
	}
}

func (m *MockSubsetKeyThresholdListener) Notify(clusterName string, key string, count int, threshold int) {
	m.events = append(m.events, MockEvent{clusterName, key, count, threshold})
}

func (m *MockSubsetKeyThresholdListener) GetEvents() []MockEvent {
	return m.events
}

type MockEvent struct {
	ClusterName string
	Key         string
	Count       int
	Threshold   int
}

func (m MockEvent) Equal(o MockEvent) bool {
	if m.ClusterName != o.ClusterName {
		return false
	}

	if m.Key != o.Key {
		return false
	}

	if m.Count != o.Count {
		return false
	}

	if m.Threshold != o.Threshold {
		return false
	}

	return true
}
