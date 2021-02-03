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

import "sync/atomic"

// BaseFeatureSpec is a basic implementation of FeatureSpec.
// Usually, a feature spec just need an init func.
type BaseFeatureSpec struct {
	DefaultValue    bool
	IsLockedDefault bool
	PreReleaseValue prerelease
	stateValue      bool  // stateValue shoule be setted by SetState
	inited          int32 // inited cannot be setted
}

func (spec *BaseFeatureSpec) Default() bool {
	return spec.DefaultValue
}

func (spec *BaseFeatureSpec) LockToDefault() bool {
	return spec.IsLockedDefault
}

func (spec *BaseFeatureSpec) SetState(enable bool) {
	spec.stateValue = enable
	atomic.StoreInt32(&spec.inited, 1)
}

func (spec *BaseFeatureSpec) State() bool {
	if atomic.LoadInt32(&spec.inited) == 0 { // not init
		return spec.DefaultValue
	}
	return spec.stateValue
}

func (spec *BaseFeatureSpec) PreRelease() prerelease {
	return spec.PreReleaseValue
}

func (spec *BaseFeatureSpec) InitFunc() {
}
