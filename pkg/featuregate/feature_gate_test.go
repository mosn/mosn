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
	"reflect"
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
	fg.FinallyInitFunc()
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
	// empty will be ignore
	p := "feature_set_to_true=true, feature_set_to_false=false ,"
	if err := fg.Set(p); err != nil {
		t.Fatal(err)
	}
	// set invalid parameters, ignore all of them
	for _, invalid := range []string{
		"feature_set_to_true,feature_set_to_false", // no equals
		"feature_set_to_false=yes",                 // not a boolean
		"feature_set_to_false==true",               // too many equals
		"feature_not_exists=true",                  // not a known feature
		"feature_lock_to_true=false",               // a locked feature
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

func TestKnownFeatures(t *testing.T) {
	fg := NewFeatureGate()
	for _, fn := range []Feature{
		"f1", "f2", "f3", "f4", "f5",
	} {
		fg.AddFeatureSpec(fn, &BaseFeatureSpec{})
	}
	// set
	fg.Set("f1=true, f2=false, f3=true, f4=false")
	known := fg.KnownFeatures()
	// verify
	expected := map[string]bool{
		"f1": true,
		"f2": false,
		"f3": true,
		"f4": false,
		"f5": false,
	}
	if !reflect.DeepEqual(expected, known) {
		t.Errorf("known features returns unexpected, known: %v, expected: %v", known, expected)
	}

}

func TestSetFromMap(t *testing.T) {
	fg := NewFeatureGate()
	features := []struct {
		name     string
		spec     FeatureSpec
		expected bool
	}{
		{
			name:     "foo",
			spec:     &BaseFeatureSpec{},
			expected: true,
		},
		{
			name:     "bar",
			spec:     &BaseFeatureSpec{},
			expected: true,
		},
	}

	m := make(map[string]bool)
	for _, f := range features {
		fg.AddFeatureSpec(Feature(f.name), f.spec)
		m[f.name] = f.expected
	}

	// expect no error setting from map.
	if err := fg.SetFromMap(m); err != nil {
		t.Errorf("error setting feature states from map: %v", err)
	}

	// expect actual state equals to expected state.
	for k, v := range m {
		if state := fg.Enabled(Feature(k)); state != v {
			t.Errorf("feature state not equal, expect: %v, actual: %v", v, state)
		}
	}

	// expect error when setting state for a non-registered feature.
	m["feature-not-found"] = true
	if err := fg.SetFromMap(m); err == nil {
		t.Errorf("setting feature state without registering, expect err, got nil.")
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
	fg.FinallyInitFunc()
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
	fg.FinallyInitFunc()
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

type _mockSpec struct {
	*BaseFeatureSpec
	f func()
}

func (sp *_mockSpec) InitFunc() {
	if sp.f != nil {
		sp.f()
	}
}

func TestFeaturGateExecuteFunc(t *testing.T) {
	fg := NewFeatureGate()
	called := []Feature{}
	for _, c := range []struct {
		key     Feature
		enabled bool
	}{
		{Feature("test1"), true},
		{Feature("test2"), true},
		{Feature("test3"), false},
		{Feature("test4"), true},
	} {
		key := c.key
		fg.AddFeatureSpec(key, &_mockSpec{
			BaseFeatureSpec: &BaseFeatureSpec{
				DefaultValue: c.enabled,
			},
			f: func() {
				called = append(called, key)
			},
		})
	}
	invalidSub := Feature("test5")
	fg.AddFeatureSpec(invalidSub, &_mockSpec{
		BaseFeatureSpec: &BaseFeatureSpec{
			DefaultValue: true,
		},
		f: func() {
			called = append(called, invalidSub)
			if ok, err := fg.Subscribe(Feature("test1"), 0); ok || err != ErrSubStage {
				t.Errorf("subscribe can only be called in finally stage, but not: %v", err)
			}
		},
	})
	fg.ExecuteInitFunc(Feature("test1"), Feature("test3"), Feature("test5"))
	// veirfy
	if !(len(called) == 2 &&
		called[0] == Feature("test1") &&
		called[1] == Feature("test5")) {
		t.Fatalf("feature spec executed is not expected: %v", called)
	}
	fg.FinallyInitFunc()
	if err := fg.WaitInitFinsh(); err != nil {
		t.Fatalf("wait finish error: %v", err)
	}
	if len(called) != 4 {
		t.Fatalf("feature spec executed is not expected: %v", called)
	}
	for _, c := range called {
		if c == Feature("test3") {
			t.Errorf("test3 is not enable, should not be called")
		}
	}
}
