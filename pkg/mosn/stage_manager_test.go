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

package mosn

import (
	"sync"
	"testing"

	"github.com/urfave/cli"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
)

func TestStageManager(t *testing.T) {
	stm := NewStageManager(&cli.Context{}, "")
	// test for mock
	testCall := 0
	configmanager.RegisterConfigLoadFunc(func(p string) *v2.MOSNConfig {
		if testCall != 2 {
			t.Errorf("load config is not called after 2 stages registered")
		}
		testCall = 0
		return &v2.MOSNConfig{}
	})
	defer configmanager.RegisterConfigLoadFunc(configmanager.DefaultConfigLoad)
	stm.newMosn = func(c *v2.MOSNConfig) *Mosn {
		testCall = 0
		return &Mosn{
			wg:     sync.WaitGroup{},
			Config: c,
		}
	}

	stm.AppendParamsParsedStage(func(_ *cli.Context) {
		testCall++
		if testCall != 1 {
			t.Errorf("unexpected params parsed stage call: 1")
		}
	}).AppendParamsParsedStage(func(_ *cli.Context) {
		testCall++
		if testCall != 2 {
			t.Errorf("unexpected params parsed stage call: 2")
		}
	}).AppendInitStage(func(_ *v2.MOSNConfig) {
		testCall++
		if testCall != 1 {
			t.Errorf("unexpected init stage call: 1")
		}
	}).AppendPreStartStage(func(_ *Mosn) {
		testCall++
		if testCall != 1 {
			t.Errorf("pre start stage call: 1")
		}
	}).AppendStartStage(func(_ *Mosn) {
		testCall++
		if testCall != 2 {
			t.Errorf("start stage call: 2")
		}
	}).AppendAfterStartStage(func(_ *Mosn) {
		testCall++
		if testCall != 3 {
			t.Errorf("after start stage call: %v", testCall)
		}
	}).AppendAfterStopStage(func(_ *Mosn) {
		testCall++
		if testCall != 4 {
			t.Errorf("after stage stage call: 3")
		}
	})
	if testCall != 0 {
		t.Errorf("should call nothing")
	}
	stm.Run()
	if !(testCall == 3 &&
		stm.data.mosn != nil &&
		stm.data.config != nil) {
		t.Errorf("stage manager runs failed...")
	}
	stm.data.mosn.Close()
	stm.WaitFinish()
	if !(testCall == 3 &&
		stm.data.mosn != nil &&
		stm.data.config != nil) {
		t.Errorf("WaitFinish runs failed...")
	}
	stm.Stop()
	if !(testCall == 4 &&
		stm.data.mosn != nil &&
		stm.data.config != nil) {
		t.Errorf("Stop runs failed...")
	}
}
