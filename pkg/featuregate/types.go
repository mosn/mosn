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
	"errors"
)

type Feature string

type prerelease string

const (
	// Values for PreRelease.
	Alpha = prerelease("ALPHA")
	Beta  = prerelease("BETA")
	GA    = prerelease("")
)

type FeatureSpec interface {
	// Default is the default enablement state for the feature
	Default() bool
	// LockToDefault indicates that the feature is locked to its default and cannot be changed
	LockToDefault() bool
	// SetState sets the enablement state for the feature
	SetState(enable bool)
	// State indicates the feature enablement
	State() bool
	// InitFunc used to init process when StartInit is invoked
	InitFunc()
	// PreRelease indicates the maturity level of the feature
	PreRelease() prerelease
}

var (
	ErrInited    = errors.New("feature gate is already inited")
	ErrNotInited = errors.New("feature gate is not inited")
)
