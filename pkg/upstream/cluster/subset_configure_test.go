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
