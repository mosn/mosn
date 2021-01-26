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

package featuregate

import (
	"time"
)

var defaultFeatureGate *FeatureGate = NewFeatureGate()

func Enabled(key Feature) bool {
	return defaultFeatureGate.Enabled(key)
}

func Subscribe(key Feature, timeout time.Duration) (bool, error) {
	return defaultFeatureGate.Subscribe(key, timeout)
}

func Set(value string) error {
	return defaultFeatureGate.Set(value)
}

func SetFromMap(m map[string]bool) error {
	return defaultFeatureGate.SetFromMap(m)
}

func AddFeatureSpec(key Feature, spec FeatureSpec) error {
	return defaultFeatureGate.AddFeatureSpec(key, spec)
}

func SetFeatureState(key Feature, enable bool) error {
	return defaultFeatureGate.SetFeatureState(key, enable)
}

func KnownFeatures() map[string]bool {
	return defaultFeatureGate.KnownFeatures()
}

func StartInit() {
	defaultFeatureGate.StartInit()
}

func WaitInitFinsh() error {
	return defaultFeatureGate.WaitInitFinsh()
}
