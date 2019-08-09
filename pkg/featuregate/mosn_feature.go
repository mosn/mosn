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

const (
	// owner: @zewen
	// alpha:
	// beta:
	XdsMtlsEnable Feature = "XdsMtlsEnable"

	// owner @zewen
	// alpha:
	// beta:
	PayLoadLimitEnable Feature = "PayLoadLimitEnable"

	// owner: @zewen
	// alpha:
	// beta:
	MultiTenantMode Feature = "MultiTenantMode"
)

var (
	// DefaultMutableFeatureGate is a mutable version of DefaultFeatureGate.
	// Only top-level commands/options setup should make use of this.
	DefaultMutableFeatureGate MutableFeatureGate = NewFeatureGate()

	// DefaultFeatureGate is a shared global FeatureGate.
	// Top-level commands/options setup that needs to modify this feature gate should use DefaultMutableFeatureGate.
	DefaultFeatureGate FeatureGate = DefaultMutableFeatureGate
)

func init() {
	if err := DefaultMutableFeatureGate.Add(defaultMosnFeatureGates); err != nil {
		panic(err)
	}
}

// defaultMosnFeatureGates consists of all known Mosn-specific feature keys.
// To add a new feature, define a key for it above and add it here.
// The features will be available throughout mosn binaries.
var defaultMosnFeatureGates = map[Feature]FeatureSpec{
	XdsMtlsEnable:      {Default: false, PreRelease: Alpha},
	PayLoadLimitEnable: {Default: false, PreRelease: Alpha},
	MultiTenantMode:    {Default: false, PreRelease: Alpha},
}
