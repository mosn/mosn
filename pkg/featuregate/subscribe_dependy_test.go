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
	"testing"
	"time"
)

const (
	_TestFeatureA Feature = "feature A"
	_TestFeatureB Feature = "feature B"
)

// Subscribe with update to ready can be used in feature dependcy.
// for example, feature A maybe needs some features from feature B, so the feature A should wait feature B finished before A start
type _FeatureA struct {
	*BaseFeatureSpec
	mode int
	t    *testing.T
}

// feature A depends on feature B, so feature A subscribe feature B
func (fa *_FeatureA) InitFunc() {
	ready, err := Subscribe(_TestFeatureB, 5*time.Second)
	if err != nil {
		fa.t.Fatalf("subscribe feature B failed: %v", err)
	}
	if ready {
		fa.mode = 1
	} else {
		fa.mode = 2
	}

}

type _FeatureB struct {
	*BaseFeatureSpec
	notify chan struct{}
}

func (fb *_FeatureB) SendNotify() {
	close(fb.notify)
}

func (fb *_FeatureB) InitFunc() {
	<-fb.notify
}

// just for test
func _ResetDefaultFeatureGate() {
	defaultFeatureGate = NewFeatureGate()
}

func TestSubDependcyReady(t *testing.T) {
	_ResetDefaultFeatureGate()
	fa := &_FeatureA{
		BaseFeatureSpec: &BaseFeatureSpec{},
		t:               t,
	}
	fb := &_FeatureB{
		BaseFeatureSpec: &BaseFeatureSpec{},
		notify:          make(chan struct{}),
	}
	AddFeatureSpec(_TestFeatureA, fa)
	AddFeatureSpec(_TestFeatureB, fb)
	Set("feature A=true, feature B=true")
	FinallyInitFunc()
	// feature B not ready, feature A sub will timeout
	if _, err := Subscribe(_TestFeatureA, time.Second); !_IsSubTimeout(err) {
		t.Errorf("expected sub feature A timeout, but got: %v", err)
	}
	fb.SendNotify() // feature B ready
	ready, err := Subscribe(_TestFeatureA, time.Second)
	if err != nil || !ready {
		t.Errorf("feature A, ready:%v, error: %v", ready, err)
	}
	if fa.mode != 1 {
		t.Error("feature a mode is not expected")
	}

}

func TestSubDependcyNotReady(t *testing.T) {
	_ResetDefaultFeatureGate()
	fa := &_FeatureA{
		BaseFeatureSpec: &BaseFeatureSpec{},
		t:               t,
	}
	fb := &_FeatureB{
		BaseFeatureSpec: &BaseFeatureSpec{},
		notify:          make(chan struct{}),
	}
	AddFeatureSpec(_TestFeatureA, fa)
	AddFeatureSpec(_TestFeatureB, fb)
	Set("feature A=true, feature B=false")
	FinallyInitFunc()
	// feature B is not ready, so the feature A is run as 2 mode
	ready, err := Subscribe(_TestFeatureA, time.Second)
	if err != nil || !ready {
		t.Errorf("feature A, ready:%v, error: %v", ready, err)
	}
	if fa.mode != 2 {
		t.Error("feature a mode is not expected")
	}
}

func TestSubDependcyNotReady2(t *testing.T) {
	_ResetDefaultFeatureGate()
	fa := &_FeatureA{
		BaseFeatureSpec: &BaseFeatureSpec{},
		t:               t,
	}
	fb := &_FeatureB{
		BaseFeatureSpec: &BaseFeatureSpec{},
		notify:          make(chan struct{}),
	}
	AddFeatureSpec(_TestFeatureA, fa)
	AddFeatureSpec(_TestFeatureB, fb)
	Set("feature A=false, feature B=true")
	FinallyInitFunc()
	fb.SendNotify() // feature B ready
	ready, err := Subscribe(_TestFeatureA, time.Second)
	if err != nil || ready {
		t.Errorf("feature A, ready:%v, error: %v", ready, err)
	}
	if fa.mode != 0 {
		t.Error("feature a mode is not expected")
	}

}
