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

package stagemanager

import (
	"sync/atomic"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/urfave/cli"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/mock"
)

func testSetGetState(state State, t *testing.T) {
	t.Helper()
	stm.SetState(state)
	s := GetState()
	if s != state {
		t.Errorf("get after set, state expect %v while got %v", state, s)
	}
}

var allStates = []State{ParamsParsed, Initing, PreStart, Starting, AfterStart, Running, GracefulStopping, Stopping, AfterStop, Stopped}

func TestSetGetState(t *testing.T) {
	state := GetState()
	if state != Nil {
		t.Errorf("unexpected init state, got: %v", state)
	}

	for _, s := range allStates {
		testSetGetState(s, t)
	}
}

func TestOnStateChanged(t *testing.T) {
	calls := map[State]uint32{}
	onChanged := func(s State) {
		cnt, ok := calls[s]
		if ok {
			calls[s] = atomic.AddUint32(&cnt, 1)
		}
	}
	RegisterOnStateChanged(onChanged)
	for _, s := range allStates {
		calls[s] = 0
	}
	for _, s := range allStates {
		for i := 0; i < 10; i++ {
			stm.SetState(s)
		}
	}
	for s, cnt := range calls {
		if cnt != 10 {
			t.Errorf("state:%d count is %d", s, cnt)
		}
	}
}

func TestStageManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := mock.NewMockApplication(ctrl)

	app.EXPECT().Init(gomock.Any()).Return(nil)
	app.EXPECT().Start().Return()
	app.EXPECT().InheritConnections().Return(nil)
	app.EXPECT().Shutdown().Return(nil)
	app.EXPECT().Close(false).Return()

	stm := InitStageManager(&cli.Context{}, "", app)
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
		// testCall reset to 0 in the new registered ConfigLoad
		testCall++
		if testCall != 1 {
			t.Errorf("init stage call, expect 1 while got %v", testCall)
		}
	}).AppendPreStartStage(func(_ Application) {
		testCall++
		if testCall != 2 {
			t.Errorf("pre start stage call, expect 2 while got %v", testCall)
		}
	}).AppendStartStage(func(_ Application) {
		testCall++
		if testCall != 3 {
			t.Errorf("start stage call, expect 3 while got %v", testCall)
		}
	}).AppendAfterStartStage(func(_ Application) {
		testCall++
		if testCall != 4 {
			t.Errorf("after start stage call, expect 4 while got %v", testCall)
		}
	}).AppendBeforeStopStage(func(StopAction, Application) error {
		testCall++
		if testCall != 5 {
			t.Errorf("pre stop stage call, expect 5 while got %v", testCall)
		}
		return nil
	}).AppendGracefulStopStage(func(_ Application) error {
		testCall++
		if testCall != 6 {
			t.Errorf("pre stop stage call, expect 5 while got %v", testCall)
		}
		return nil
	}).AppendAfterStopStage(func(_ Application) {
		testCall++
		if testCall != 7 {
			t.Errorf("after stop stage call, expect 6 while got %v", testCall)
		}
	})
	if testCall != 0 {
		t.Errorf("should call nothing")
	}
	stm.Run()
	if !(testCall == 4 &&
		GetState() == Running) {
		t.Errorf("run stage failed, testCall: %v, stage: %v", testCall, GetState())
	}
	NoticeStop(GracefulStop)
	stm.WaitFinish()
	// NoticeStop will call runBeforeStopStage & WaitFinish will set the state to Running
	if !(testCall == 5 &&
		GetState() == Running) {
		t.Errorf("wait stage failed, testCall: %v, stage: %v", testCall, GetState())
	}
	stm.Stop()
	if !(testCall == 7 &&
		GetState() == Stopped) {
		t.Errorf("stop stage failed, testCall: %v, stage: %v", testCall, GetState())
	}
}
