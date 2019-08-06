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
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestFeatureGateFlag(t *testing.T) {
	// gates for testing
	const testAlphaGate Feature = "TestAlpha"
	const testBetaGate Feature = "TestBeta"

	tests := []struct {
		arg        string
		expect     map[Feature]bool
		parseError string
	}{
		{
			arg: "",
			expect: map[Feature]bool{
				allAlphaGate:  false,
				testAlphaGate: false,
				testBetaGate:  false,
			},
		},
		{
			arg: "fooBarBaz=true",
			expect: map[Feature]bool{
				allAlphaGate:  false,
				testAlphaGate: false,
				testBetaGate:  false,
			},
			parseError: "unrecognized feature gate: fooBarBaz",
		},
		{
			arg: "AllAlpha=false",
			expect: map[Feature]bool{
				allAlphaGate:  false,
				testAlphaGate: false,
				testBetaGate:  false,
			},
		},
		{
			arg: "AllAlpha=true",
			expect: map[Feature]bool{
				allAlphaGate:  true,
				testAlphaGate: true,
				testBetaGate:  false,
			},
		},
		{
			arg: "AllAlpha=banana",
			expect: map[Feature]bool{
				allAlphaGate:  false,
				testAlphaGate: false,
				testBetaGate:  false,
			},
			parseError: "invalid value of AllAlpha",
		},
		{
			arg: "AllAlpha=false,TestAlpha=true",
			expect: map[Feature]bool{
				allAlphaGate:  false,
				testAlphaGate: true,
				testBetaGate:  false,
			},
		},
		{
			arg: "TestAlpha=true,AllAlpha=false",
			expect: map[Feature]bool{
				allAlphaGate:  false,
				testAlphaGate: true,
				testBetaGate:  false,
			},
		},
		{
			arg: "AllAlpha=true,TestAlpha=false",
			expect: map[Feature]bool{
				allAlphaGate:  true,
				testAlphaGate: false,
				testBetaGate:  false,
			},
		},
		{
			arg: "TestAlpha=false,AllAlpha=true",
			expect: map[Feature]bool{
				allAlphaGate:  true,
				testAlphaGate: false,
				testBetaGate:  false,
			},
		},
		{
			arg: "TestBeta=true,AllAlpha=false",
			expect: map[Feature]bool{
				allAlphaGate:  false,
				testAlphaGate: false,
				testBetaGate:  true,
			},
		},
	}
	for i, test := range tests {
		fs := pflag.NewFlagSet("testfeaturegateflag", pflag.ContinueOnError)
		f := NewFeatureGate()
		f.Add(map[Feature]FeatureSpec{
			testAlphaGate: {Default: false, PreRelease: Alpha},
			testBetaGate:  {Default: false, PreRelease: Beta},
		})
		f.AddFlag(fs)

		err := fs.Parse([]string{fmt.Sprintf("--%s=%s", flagName, test.arg)})
		if test.parseError != "" {
			if !strings.Contains(err.Error(), test.parseError) {
				t.Errorf("%d: Parse() Expected %v, Got %v", i, test.parseError, err)
			}
		} else if err != nil {
			t.Errorf("%d: Parse() Expected nil, Got %v", i, err)
		}
		for k, v := range test.expect {
			if actual := f.enabled.Load().(map[Feature]bool)[k]; actual != v {
				t.Errorf("%d: expected %s=%v, Got %v", i, k, v, actual)
			}
		}
	}
}

func TestFeatureGateOverride(t *testing.T) {
	const testAlphaGate Feature = "TestAlpha"
	const testBetaGate Feature = "TestBeta"

	// Don't parse the flag, assert defaults are used.
	var f *defaultFeatureGate = NewFeatureGate()
	f.Add(map[Feature]FeatureSpec{
		testAlphaGate: {Default: false, PreRelease: Alpha},
		testBetaGate:  {Default: false, PreRelease: Beta},
	})

	f.Set("TestAlpha=true,TestBeta=true")
	if f.Enabled(testAlphaGate) != true {
		t.Errorf("Expected true")
	}
	if f.Enabled(testBetaGate) != true {
		t.Errorf("Expected true")
	}

	f.Set("TestAlpha=false")
	if f.Enabled(testAlphaGate) != false {
		t.Errorf("Expected false")
	}
	if f.Enabled(testBetaGate) != true {
		t.Errorf("Expected true")
	}
}

func TestFeatureGateFlagDefaults(t *testing.T) {
	// gates for testing
	const testAlphaGate Feature = "TestAlpha"
	const testBetaGate Feature = "TestBeta"

	// Don't parse the flag, assert defaults are used.
	var f *defaultFeatureGate = NewFeatureGate()
	f.Add(map[Feature]FeatureSpec{
		testAlphaGate: {Default: false, PreRelease: Alpha},
		testBetaGate:  {Default: true, PreRelease: Beta},
	})

	if f.Enabled(testAlphaGate) != false {
		t.Errorf("Expected false")
	}
	if f.Enabled(testBetaGate) != true {
		t.Errorf("Expected true")
	}
}

func TestFeatureGateKnownFeatures(t *testing.T) {
	// gates for testing
	const (
		testAlphaGate      Feature = "TestAlpha"
		testBetaGate       Feature = "TestBeta"
		testGAGate         Feature = "TestGA"
		testDeprecatedGate Feature = "TestDeprecated"
	)

	// Don't parse the flag, assert defaults are used.
	var f *defaultFeatureGate = NewFeatureGate()
	f.Add(map[Feature]FeatureSpec{
		testAlphaGate: {Default: false, PreRelease: Alpha},
		testBetaGate:  {Default: true, PreRelease: Beta},
		testGAGate:    {Default: true, PreRelease: GA},
	})

	known := strings.Join(f.KnownFeatures(), " ")

	assert.Contains(t, known, testAlphaGate)
	assert.Contains(t, known, testBetaGate)
	assert.NotContains(t, known, testGAGate)
	assert.NotContains(t, known, testDeprecatedGate)
}

func TestFeatureGateSetFromMap(t *testing.T) {
	// gates for testing
	const testAlphaGate Feature = "TestAlpha"
	const testBetaGate Feature = "TestBeta"
	const testLockedTrueGate Feature = "TestLockedTrue"
	const testLockedFalseGate Feature = "TestLockedFalse"

	tests := []struct {
		name        string
		setmap      map[string]bool
		expect      map[Feature]bool
		setmapError string
	}{
		{
			name: "set TestAlpha and TestBeta true",
			setmap: map[string]bool{
				"TestAlpha": true,
				"TestBeta":  true,
			},
			expect: map[Feature]bool{
				testAlphaGate: true,
				testBetaGate:  true,
			},
		},
		{
			name: "set TestBeta true",
			setmap: map[string]bool{
				"TestBeta": true,
			},
			expect: map[Feature]bool{
				testAlphaGate: false,
				testBetaGate:  true,
			},
		},
		{
			name: "set TestAlpha false",
			setmap: map[string]bool{
				"TestAlpha": false,
			},
			expect: map[Feature]bool{
				testAlphaGate: false,
				testBetaGate:  false,
			},
		},
		{
			name: "set TestInvaild true",
			setmap: map[string]bool{
				"TestInvaild": true,
			},
			expect: map[Feature]bool{
				testAlphaGate: false,
				testBetaGate:  false,
			},
			setmapError: "unrecognized feature gate:",
		},
		{
			name: "set locked gates",
			setmap: map[string]bool{
				"TestLockedTrue":  true,
				"TestLockedFalse": false,
			},
			expect: map[Feature]bool{
				testAlphaGate: false,
				testBetaGate:  false,
			},
		},
		{
			name: "set locked gates",
			setmap: map[string]bool{
				"TestLockedTrue": false,
			},
			expect: map[Feature]bool{
				testAlphaGate: false,
				testBetaGate:  false,
			},
			setmapError: "cannot set feature gate TestLockedTrue to false, feature is locked to true",
		},
		{
			name: "set locked gates",
			setmap: map[string]bool{
				"TestLockedFalse": true,
			},
			expect: map[Feature]bool{
				testAlphaGate: false,
				testBetaGate:  false,
			},
			setmapError: "cannot set feature gate TestLockedFalse to true, feature is locked to false",
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("SetFromMap %s", test.name), func(t *testing.T) {
			f := NewFeatureGate()
			f.Add(map[Feature]FeatureSpec{
				testAlphaGate:       {Default: false, PreRelease: Alpha},
				testBetaGate:        {Default: false, PreRelease: Beta},
				testLockedTrueGate:  {Default: true, PreRelease: GA, LockToDefault: true},
				testLockedFalseGate: {Default: false, PreRelease: GA, LockToDefault: true},
			})
			err := f.SetFromMap(test.setmap)
			if test.setmapError != "" {
				if err == nil {
					t.Errorf("expected error, got none")
				} else if !strings.Contains(err.Error(), test.setmapError) {
					t.Errorf("%d: SetFromMap(%#v) Expected err:%v, Got err:%v", i, test.setmap, test.setmapError, err)
				}
			} else if err != nil {
				t.Errorf("%d: SetFromMap(%#v) Expected success, Got err:%v", i, test.setmap, err)
			}
			for k, v := range test.expect {
				if actual := f.Enabled(k); actual != v {
					t.Errorf("%d: SetFromMap(%#v) Expected %s=%v, Got %s=%v", i, test.setmap, k, v, k, actual)
				}
			}
		})
	}
}

func TestFeatureGateString(t *testing.T) {
	// gates for testing
	const testAlphaGate Feature = "TestAlpha"
	const testBetaGate Feature = "TestBeta"
	const testGAGate Feature = "TestGA"

	featuremap := map[Feature]FeatureSpec{
		testGAGate:    {Default: true, PreRelease: GA},
		testAlphaGate: {Default: false, PreRelease: Alpha},
		testBetaGate:  {Default: true, PreRelease: Beta},
	}

	tests := []struct {
		setmap map[string]bool
		expect string
	}{
		{
			setmap: map[string]bool{
				"TestAlpha": false,
			},
			expect: "TestAlpha=false",
		},
		{
			setmap: map[string]bool{
				"TestAlpha": false,
				"TestBeta":  true,
			},
			expect: "TestAlpha=false,TestBeta=true",
		},
		{
			setmap: map[string]bool{
				"TestGA":    true,
				"TestAlpha": false,
				"TestBeta":  true,
			},
			expect: "TestAlpha=false,TestBeta=true,TestGA=true",
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("SetFromMap %s", test.expect), func(t *testing.T) {
			f := NewFeatureGate()
			f.Add(featuremap)
			f.SetFromMap(test.setmap)
			result := f.String()
			if result != test.expect {
				t.Errorf("%d: SetFromMap(%#v) Expected %s, Got %s", i, test.setmap, test.expect, result)
			}
		})
	}
}

func TestFeatureGateReady(t *testing.T) {
	const testDisabled Feature = "testDisabled"
	const testDisabledWithSubmoduleNoReady Feature = "testDisabledWithSubmoduleNoReady"
	const testDisabledWithSubmoduleReady Feature = "testDisabledWithSubmoduleReady"
	const testDisabledWithSubmodule Feature = "testDisabledWithSubmodule"

	const testEnabled Feature = "testEnabled"
	const testEnabledWithSubmoduleNoReady Feature = "testEnabledWithSubmoduleNoReady"
	const testEnabledWithSubmoduleReady Feature = "testEnabledWithSubmoduleReady"
	const testEnabledWithSubmodule Feature = "testEnabledWithSubmodule"

	// Don't parse the flag, assert defaults are used.
	var f = NewFeatureGate()
	f.Add(map[Feature]FeatureSpec{
		testDisabled:                     {Default: false, PreRelease: Beta, Submodules: NilSubmodule},
		testDisabledWithSubmoduleNoReady: {Default: false, PreRelease: Beta, Submodules: []Submodule{"submodule001"}},
		testDisabledWithSubmoduleReady:   {Default: false, PreRelease: Beta, Submodules: []Submodule{"submodule011"}},
		testDisabledWithSubmodule:        {Default: false, PreRelease: Beta, Submodules: []Submodule{"submodule021", "submodule022"}},

		testEnabled:                     {Default: true, PreRelease: Beta, Submodules: NilSubmodule},
		testEnabledWithSubmoduleNoReady: {Default: true, PreRelease: Beta, Submodules: []Submodule{"submodule031"}},
		testEnabledWithSubmoduleReady:   {Default: true, PreRelease: Beta, Submodules: []Submodule{"submodule041"}},
		testEnabledWithSubmodule:        {Default: true, PreRelease: Beta, Submodules: []Submodule{"submodule051", "submodule052"}},
	})

	f.UpdateToReady(testDisabledWithSubmoduleReady, "submodule011")
	f.UpdateToReady(testDisabledWithSubmodule, "submodule021")
	f.UpdateToReady(testEnabledWithSubmoduleReady, "submodule041")
	f.UpdateToReady(testEnabledWithSubmodule, "submodule051")

	if f.IsReady(testDisabled) {
		t.Errorf("Excepted false")
	}
	if f.IsReady(testDisabledWithSubmoduleNoReady) {
		t.Errorf("Excepted false")
	}
	if f.IsReady(testDisabledWithSubmoduleReady) {
		t.Errorf("Excepted false")
	}
	if f.IsReady(testDisabledWithSubmoduleReady) {
		t.Errorf("Excepted false")
	}

	if !f.IsReady(testEnabled) {
		t.Errorf("Excepted true")
	}
	if f.IsReady(testEnabledWithSubmoduleNoReady) {
		t.Errorf("Excepted false")
	}
	if !f.IsReady(testEnabledWithSubmoduleReady) {
		t.Errorf("Excepted true")
	}
	if f.IsReady(testEnabledWithSubmodule) {
		t.Errorf("Excepted false")
	}
}

func TestFeatureGateUpdateToReady(t *testing.T) {
	const testDisabled Feature = "testDisabled"
	const testEnabledWithMissingSubmodule Feature = "testEnabledWithMissingSubmodule"
	const testEnabledWithRepeatUpdate Feature = "testEnabledWithRepeatUpdate"
	const testEnabledSuccessfully Feature = "testEnabledSuccessfully"

	// Don't parse the flag, assert defaults are used.
	var f = NewFeatureGate()
	f.Add(map[Feature]FeatureSpec{
		testDisabled:                    {Default: false, PreRelease: Beta, Submodules: NilSubmodule},
		testEnabledWithMissingSubmodule: {Default: true, PreRelease: Beta, Submodules: []Submodule{"existSubmodule"}},
		testEnabledWithRepeatUpdate:     {Default: true, PreRelease: Beta, Submodules: []Submodule{"submodule001"}},
		testEnabledSuccessfully:         {Default: true, PreRelease: Beta, Submodules: []Submodule{"submodule021", "submodule022"}},
	})

	var err error
	err = f.UpdateToReady(testDisabled, "nothing")
	if err == nil || err.Error() != fmt.Sprintf("feature %s is disabled, can not update %s to ready ", testDisabled, "nothing") {
		t.Errorf("Excepted error: feature %s is disabled, can not update %s to ready ", testDisabled, "nothing")
	}

	err = f.UpdateToReady(testEnabledWithMissingSubmodule, "missingSubmodule")
	if err == nil || err.Error() != fmt.Sprintf("submodule %s.%s unknow", testEnabledWithMissingSubmodule, "missingSubmodule") {
		t.Errorf("Excepted error: submodule %s.%s unknow", testEnabledWithMissingSubmodule, "missingSubmodule")
	}

	err = f.UpdateToReady(testEnabledWithRepeatUpdate, "submodule001")
	if err != nil {
		t.Error("Excepted no error")
	}
	err = f.UpdateToReady(testEnabledWithRepeatUpdate, "submodule001")
	if err == nil || err.Error() != fmt.Sprintf("repeat setting submodule %s.%s to ready", testEnabledWithRepeatUpdate, "submodule001") {
		t.Errorf("Excepted error: repeat setting submodule %s.%s to ready", testEnabledWithRepeatUpdate, "submodule001")
	}

	err = f.UpdateToReady(testEnabledSuccessfully, "submodule021")
	if err != nil {
		t.Error("Excepted no error")
	}
	if f.IsReady(testEnabledSuccessfully) {
		t.Error("Excepted false")
	}
	err = f.UpdateToReady(testEnabledSuccessfully, "submodule022")
	if err != nil {
		t.Error("Excepted no error")
	}
	if !f.IsReady(testEnabledSuccessfully) {
		t.Error("Excepted true")
	}

}

func TestFeatureGateUpdateToSubscribe(t *testing.T) {
	const testDisabled Feature = "testDisabled"
	const testMissingSubmodule Feature = "testMissingSubmodule"
	const testAfterReady Feature = "testAfterReady"
	const testBeforeReady Feature = "testBeforeReady"

	// Don't parse the flag, assert defaults are used.
	var f = NewFeatureGate()
	f.Add(map[Feature]FeatureSpec{
		testDisabled:         {Default: false, PreRelease: Beta, Submodules: NilSubmodule},
		testMissingSubmodule: {Default: false, PreRelease: Beta, Submodules: []Submodule{"existSubmodule"}},
		testAfterReady:       {Default: true, PreRelease: Beta, Submodules: []Submodule{"submodule001"}},
		testBeforeReady:      {Default: true, PreRelease: Beta, Submodules: []Submodule{"submodule011"}},
	})

	var err error
	_, err = f.Subscribe("missingFeatureKey", "")
	if err == nil {
		t.Errorf("Excepted error: subscribe fails, make sure %s/%s is inited correctly", "missingFeatureKey", "")
	}

	_, err = f.Subscribe(testDisabled, "")
	if err == nil {
		t.Errorf("Excepted error: subscribe fails, make sure %s/%s is inited correctly", testDisabled, "")
	}

	_, err = f.Subscribe(testMissingSubmodule, "missSubmodule")
	if err == nil {
		t.Errorf("Excepted error: subscribe fails, make sure %s/%s is inited correctly", testMissingSubmodule, "missSubmodule")
	}

	f.UpdateToReady(testAfterReady, "submodule001")
	msg, err := f.Subscribe(testAfterReady, "submodule001")
	if err != nil || !msg.Ready {
		t.Errorf("Excepted not error, and %s.%s is ready", testAfterReady, "submodule001")
	}

	msg, err = f.Subscribe(testBeforeReady, "submodule011")
	if err != nil || msg.Ready {
		t.Errorf("Excepted not error, and %s.%s is not ready", testAfterReady, "submodule001")
	}
	go f.UpdateToReady(testBeforeReady, "submodule011")
	c, _ := <-msg.Channel
	if !c.Ready {
		t.Errorf("Excepted %s.%s is ready", testBeforeReady, "submodule011")
	}
}
