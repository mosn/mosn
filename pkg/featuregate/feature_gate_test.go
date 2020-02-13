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
	"strings"
	"testing"
	"time"
)

func TestAddAndSetFeatureSpec(t *testing.T) {
	var feature Feature = "test"
	var lock Feature = "lockToTrue"
	fg := NewFeatureGate()
	fg.AddFeatureSpec(feature, &BaseFeatureSpec{
		DefaultValue: true,
	})
	fg.SetFeatureState(feature, false)
	if fg.Enabled(feature) {
		t.Error("set feature to false, but got true")
	}
	fg.AddFeatureSpec(lock, &BaseFeatureSpec{
		DefaultValue:    true,
		IsLockedDefault: true,
	})
	// duplicate add, should be ignore
	fg.AddFeatureSpec(lock, &BaseFeatureSpec{})
	// the first add is valid, lock to default
	fg.SetFeatureState(lock, false)
	if !fg.Enabled(lock) {
		t.Error("feature should always to be true")
	}
	fg.StartInit()
	fg.WaitInitFinsh()
	var failed Feature = "failed"
	if err := fg.AddFeatureSpec(failed, &BaseFeatureSpec{}); err == nil {
		t.Error("add a new feature after init should be failed")
	}
	if err := fg.SetFeatureState(feature, true); err == nil {
		t.Error("feature state should not be setted after init")
	}
}

func TestSetFeatureGate(t *testing.T) {
	fg := NewFeatureGate()
	// if feature is not set, use default value
	// if feature is not lock to default, use the set value
	// if feature is lock to default, ignore the set value
	features := []struct {
		name     Feature
		spec     FeatureSpec
		expected bool
	}{
		{
			name:     Feature("default_false"),
			spec:     &BaseFeatureSpec{},
			expected: false,
		},
		{
			name:     Feature("default_true"),
			spec:     &BaseFeatureSpec{DefaultValue: true},
			expected: true,
		},
		{
			name:     Feature("feature_set_to_true"),
			spec:     &BaseFeatureSpec{},
			expected: true,
		},
		{
			name:     Feature("feature_set_to_false"),
			spec:     &BaseFeatureSpec{DefaultValue: true},
			expected: false,
		},
		{
			name:     Feature("feature_lock_to_true"),
			spec:     &BaseFeatureSpec{DefaultValue: true, IsLockedDefault: true},
			expected: true,
		},
	}
	for _, fc := range features {
		fg.AddFeatureSpec(fc.name, fc.spec)
	}
	// set only one feature
	one := "feature_set_to_true=true"
	fg.Set(one)
	if !fg.Enabled("feature_set_to_true") {
		t.Error("set feature failed")
	}
	// if exists a feature not registered, ignore it
	// empty will be ignore
	p := "feature_set_to_true=true, feature_set_to_false=false, feature_lock_to_true=false, feature_not_exists=true,,"
	fg.Set(p)
	// set invalid parameters, ignore all of them
	for _, invalid := range []string{
		"feature_set_to_true,feature_set_to_false", // no equals
		"feature_set_to_false=yes",                 // not a boolean
		"feature_set_to_false==true",               // too many equals
	} {
		if err := fg.Set(invalid); err == nil {
			t.Error("expected set failed")
		}
	}

	for _, fc := range features {
		if fg.Enabled(fc.name) != fc.expected {
			t.Errorf("feature %s state expected %v", fc.name, fc.expected)
		}
	}

}

func _IsSubTimeout(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "subscribe timeout")
}

type _NotifySpec struct {
	*BaseFeatureSpec
	inited bool
	ch     chan struct{}
}

func (s *_NotifySpec) InitFunc() {
	s.inited = true
	<-s.ch
}

func TestFeatureGateSubscribe(t *testing.T) {
	fg := NewFeatureGate()
	var feature Feature = "test"
	var notify Feature = "notify"
	fg.AddFeatureSpec(feature, &_NotifySpec{
		BaseFeatureSpec: &BaseFeatureSpec{},
		ch:              make(chan struct{}),
	})
	fnotify := &_NotifySpec{
		BaseFeatureSpec: &BaseFeatureSpec{},
		ch:              make(chan struct{}),
	}
	fg.AddFeatureSpec(notify, fnotify)
	fg.Set("test=false, notify=true")
	if _, err := fg.Subscribe(feature, 0); err == nil {
		t.Error("subscribe before init, expected got an error")
	}
	fg.StartInit()
	// sub a disable feature, directly return, no init func called
	ready, err := fg.Subscribe(feature, time.Second)
	if err != nil || ready {
		t.Error("expected got false directly")
	}
	if _, err := fg.Subscribe(notify, time.Second); !_IsSubTimeout(err) {
		t.Error("expected timeout")
	}
	ch := make(chan struct{})
	go func() {
		defer func() {
			close(ch)
		}()
		ready, err := fg.Subscribe(notify, 0)
		if err != nil {
			t.Errorf("subscribe failed: %v", err)
			return
		}
		if !ready {
			t.Error("subscribe expected ready")
		}
	}()
	// send notify
	close(fnotify.ch)
	select {
	case <-ch:
		if !fnotify.inited {
			t.Error("expected init notify feature")
		}
	case <-time.After(time.Second):
		t.Error("subscribe failed")
	}
}

func TestDefaultSubscribe(t *testing.T) {
	fg := NewFeatureGate()
	var notify Feature = "notify"
	var notify2 Feature = "notify2"
	fnotify := &_NotifySpec{
		BaseFeatureSpec: &BaseFeatureSpec{
			DefaultValue: true,
		},
		ch: make(chan struct{}),
	}
	fnotify2 := &_NotifySpec{
		BaseFeatureSpec: &BaseFeatureSpec{
			DefaultValue: false,
		},
		ch: make(chan struct{}),
	}
	fg.AddFeatureSpec(notify, fnotify)
	fg.AddFeatureSpec(notify2, fnotify2)
	fg.StartInit()
	if _, err := fg.Subscribe(notify, time.Second); !_IsSubTimeout(err) {
		t.Error("expected timeout")
	}
	if _, err := fg.Subscribe(notify2, time.Second); err != nil {
		t.Error("expected sub return not ready")
	}
	ch := make(chan struct{})
	go func() {
		defer func() {
			close(ch)
		}()
		ready, err := fg.Subscribe(notify, 0)
		if err != nil {
			t.Errorf("subscribe failed: %v", err)
			return
		}
		if !ready {
			t.Error("subscribe expected ready")
		}
	}()

	// send notify
	close(fnotify.ch)
	select {
	case <-ch:
		if !fnotify.inited {
			t.Error("expected init notify feature")
		}
	case <-time.After(time.Second):
		t.Error("subscribe failed")
	}
	if fnotify2.inited {
		t.Error("notify2 should not be inited")
	}
}
