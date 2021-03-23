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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"mosn.io/mosn/pkg/log"
)

type knownFeatureSpec struct {
	FeatureSpec
	once    sync.Once
	channel chan struct{}
}

// stage represents the featuregate execution stage state.
// InitFunc of feature spec is called asynchronously in the finally stage,
// otherwise it is called synchronously.
// The Subscribe can be called only in finally stage.
type stage uint8

const (
	initialize stage = iota
	finally
)

type FeatureGate struct {
	// inited indicates whether the featuregate is initialized or not.
	// feature specs cannot be added into featuregate after initialized.
	// feature specs state cannot be changed after initialized.
	inited bool
	stage  stage
	// known stores the feature specs added into the featuregate
	known map[Feature]*knownFeatureSpec
	// finshed stores the feature specs that are already inited
	finshed map[Feature]struct{}
	wg      sync.WaitGroup
	once    sync.Once
}

func NewFeatureGate() *FeatureGate {
	fg := &FeatureGate{
		known:   map[Feature]*knownFeatureSpec{},
		finshed: map[Feature]struct{}{},
		stage:   initialize,
	}
	return fg
}

// AddFeatureSpec adds feature to the featureGate.
// features should be added by AddFeatureSpec
func (fg *FeatureGate) AddFeatureSpec(key Feature, spec FeatureSpec) error {
	if fg.inited {
		log.StartLogger.Errorf("[feature gate] feature gate is already inited")
		return ErrInited
	}
	// Copy existing state
	if _, ok := fg.known[key]; ok {
		log.StartLogger.Infof("[feature gate] duplicate feature added")
		return nil
	}
	fg.known[key] = &knownFeatureSpec{
		FeatureSpec: spec,
		channel:     make(chan struct{}),
	}
	return nil
}

// Set parses and stores flag gates for known features
// from a string like feature1=true,feature2=false,...
func (fg *FeatureGate) Set(value string) error {
	if fg.inited {
		log.StartLogger.Errorf("[feature gate] feature gate is already inited")
		return ErrInited
	}
	m := make(map[string]bool)
	for _, s := range strings.Split(value, ",") {
		if len(s) == 0 {
			continue
		}
		arr := strings.SplitN(s, "=", 2)
		if len(arr) != 2 {
			return fmt.Errorf("invalid parameter %s", s)
		}
		fname := strings.TrimSpace(arr[0])
		fstate := strings.TrimSpace(arr[1])
		boolValue, err := strconv.ParseBool(fstate)
		if err != nil {
			return fmt.Errorf("invalid state value: %s, should a boolean value", fstate)
		}
		m[fname] = boolValue
	}
	return fg.SetFromMap(m)
}

// SetFromMap stores flag gates for known features from a map[string]bool or returns an error
func (fg *FeatureGate) SetFromMap(m map[string]bool) error {
	if fg.inited {
		log.StartLogger.Errorf("[feature gate] feature gate is already inited")
		return ErrInited
	}
	// set
	for k, v := range m {
		fname := Feature(k)
		if err := fg.setFeatureState(fname, v); err != nil {
			log.StartLogger.Errorf("[feature gate] error setting feature state for %s: %v", fname, err)
			return fmt.Errorf("set feature state for %s: %v", fname, err)
		}
	}
	return nil
}

// SetFeatureState sets feature enablement state for the feature
func (fg *FeatureGate) SetFeatureState(key Feature, enable bool) error {
	if fg.inited {
		log.StartLogger.Errorf("[feature gate] feature gate is already inited")
		return ErrInited
	}
	return fg.setFeatureState(key, enable)
}

// Enabled returns true if the key is enabled.
func (fg *FeatureGate) Enabled(key Feature) bool {
	if v, ok := fg.known[key]; ok {
		return v.State()
	}
	return false
}

// KnownFeatures returns a slice of strings describing the FeatureGate's known features and status.
func (fg *FeatureGate) KnownFeatures() map[string]bool {
	known := make(map[string]bool, len(fg.known))
	for k, spec := range fg.known {
		known[string(k)] = spec.State()
	}
	return known
}

// Subscribe waits the feature is ready, and returns the feature is enabled or not.
// The timeout is the max wait time duration for subscribe. If timeout is zero, means no timeout.
func (fg *FeatureGate) Subscribe(key Feature, timeout time.Duration) (bool, error) {
	if fg.stage != finally {
		return false, ErrSubStage
	}
	b, err := fg.subscribe(key)
	if err != nil {
		return false, err
	}
	if !b.State() {
		return false, nil
	}
	// wait channel return
	if timeout > 0 {
		select {
		case <-b.channel:
		case <-time.After(timeout):
			return false, fmt.Errorf("feature %s subscribe timeout %d", key, timeout)
		}
	} else {
		<-b.channel
	}
	// b.State should be true
	return b.State(), nil
}

func (fg *FeatureGate) subscribe(key Feature) (*knownFeatureSpec, error) {
	if !fg.inited {
		return nil, ErrNotInited
	}
	b, ok := fg.known[key]
	if !ok {
		return nil, fmt.Errorf("feature %s is not found", key)
	}
	return b, nil
}

// ExecuteInitFunc calls the known feature specs in order.
// When the function is called for the first time,
// set the inited flag to prevent the known feature specs changed.
func (fg *FeatureGate) ExecuteInitFunc(keys ...Feature) {
	fg.inited = true
	for _, key := range keys {
		if _, ok := fg.finshed[key]; ok {
			continue
		}
		spec, ok := fg.known[key]
		if !ok {
			continue
		}
		fg.finshed[key] = struct{}{}
		if !spec.State() {
			log.StartLogger.Warnf("[feature gate] feature %s is not enabled", key)
			continue
		}
		log.StartLogger.Infof("[feature gate] feature %s start init", key)
		spec.InitFunc()
		fg.updateToReady(key)
		log.StartLogger.Infof("[feature gate] feature %s init done", key)
	}
}

// FinallyInitFunc calls all known feature specs asynchronously.
// If a feature spec has already finished init in ExecuteInitFunc, ignore it.
func (fg *FeatureGate) FinallyInitFunc() {
	fg.once.Do(func() {
		fg.inited = true // set inited to true, maybe no ExecuteInitFunc called.
		fg.stage = finally
		for key, spec := range fg.known {
			if _, ok := fg.finshed[key]; ok {
				continue
			}
			if !spec.State() {
				log.StartLogger.Warnf("[feature gate] feature %s is not enabled", key)
				continue
			}
			fg.wg.Add(1)
			log.StartLogger.Infof("[feature gate] feature %s start init", key)
			// do not recover, if feature gate init failed, should be panic
			go func(key Feature, spec FeatureSpec) {
				defer fg.wg.Done()
				spec.InitFunc()
				fg.updateToReady(key)
				log.StartLogger.Infof("[feature gate] feature %s init done", key)
			}(key, spec)
		}
	})
}

func (fg *FeatureGate) WaitInitFinsh() error {
	if !fg.inited {
		return ErrNotInited
	}
	fg.wg.Wait()
	return nil
}

// internal function call, the lock is called in export function
func (fg *FeatureGate) setFeatureState(key Feature, enable bool) error {
	if fg.inited {
		return ErrInited
	}
	spec, ok := fg.known[key]
	if !ok {
		return fmt.Errorf("feature %s is not registered", key)
	}
	if spec.LockToDefault() && spec.Default() != enable {
		return fmt.Errorf("feature %s is locked to %v", key, spec.Default())
	}
	spec.SetState(enable)
	return nil
}

func (fg *FeatureGate) updateToReady(key Feature) {
	if b, ok := fg.known[key]; ok {
		b.once.Do(func() {
			log.DefaultLogger.Infof("[feature gate] feature %s update to ready", key)
			close(b.channel)
		})
	}
	return
}
