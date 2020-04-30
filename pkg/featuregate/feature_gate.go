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

type FeatureGate struct {
	// lock guards writes to known, enabled, and reads/writes of closed
	lock  sync.Mutex
	known map[Feature]*knownFeatureSpec
	// inited is set to true when StartInit is called.
	inited bool
	wg     sync.WaitGroup
	once   sync.Once
}

func NewFeatureGate() *FeatureGate {
	fg := &FeatureGate{
		known: map[Feature]*knownFeatureSpec{},
	}
	return fg
}

// AddFeatureSpec adds feature to the featureGate.
// features should be added by AddFeatureSpec
func (fg *FeatureGate) AddFeatureSpec(key Feature, spec FeatureSpec) error {
	fg.lock.Lock()
	defer fg.lock.Unlock()
	if fg.inited {
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
	fg.lock.Lock()
	defer fg.lock.Unlock()
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
	fg.lock.Lock()
	defer fg.lock.Unlock()
	if fg.inited {
		log.StartLogger.Errorf("[feature gate] feature gate is already inited")
		return ErrInited
	}
	return fg.setFeatureState(key, enable)
}

// Enabled returns true if the key is enabled.
func (fg *FeatureGate) Enabled(key Feature) bool {
	fg.lock.Lock()
	defer fg.lock.Unlock()
	if v, ok := fg.known[key]; ok {
		return v.State()
	}
	return false
}

// KnownFeatures returns a slice of strings describing the FeatureGate's known features and status.
func (fg *FeatureGate) KnownFeatures() map[string]bool {
	fg.lock.Lock()
	defer fg.lock.Unlock()
	known := make(map[string]bool, len(fg.known))
	for k, spec := range fg.known {
		known[string(k)] = spec.State()
	}
	return known
}

// Subscribe waits the feature is ready, and returns the feature is enabled or not.
// The timeout is the max wait time duration for subscribe. If timeout is zero, means no timeout.
func (fg *FeatureGate) Subscribe(key Feature, timeout time.Duration) (bool, error) {
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
	fg.lock.Lock()
	defer fg.lock.Unlock()
	if !fg.inited {
		return nil, ErrNotInited
	}
	b, ok := fg.known[key]
	if !ok {
		return nil, fmt.Errorf("feature %s is not found", key)
	}
	return b, nil
}

// StartInit triggers init functions of feature
func (fg *FeatureGate) StartInit() {
	fg.lock.Lock()
	defer fg.lock.Unlock()
	fg.once.Do(func() {
		for key, spec := range fg.known {
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
	fg.inited = true
	return
}

func (fg *FeatureGate) WaitInitFinsh() error {
	fg.lock.Lock()
	defer fg.lock.Unlock()
	if !fg.inited {
		return ErrNotInited
	}
	fg.wg.Wait()
	return nil
}

// internal function call, the lock is called in export function
func (fg *FeatureGate) setFeatureState(key Feature, enable bool) error {
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
