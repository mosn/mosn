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

package faultinject

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"testing"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mlog "mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/pkg/log"
)

type mockLogger struct {
	log.ContextLogger
}

func (l *mockLogger) GetLogLevel() log.Level {
	return log.DEBUG
}

func (l *mockLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	// ingore
}

func TestMain(m *testing.M) {
	// mock debug log
	old := mlog.Proxy
	mlog.Proxy = &mockLogger{
		ContextLogger: old,
	}
	defer func() {
		mlog.Proxy = old
	}()
	os.Exit(m.Run())
}

func TestMatchUpstream(t *testing.T) {
	faultUpstream := "fault_upstream"
	testCases := []struct {
		rule     *mockRouteRule
		expected bool
	}{
		{
			rule: &mockRouteRule{
				clustername: faultUpstream,
			},
			expected: true,
		},
		{
			rule: &mockRouteRule{
				clustername: "not_matched",
			},
			expected: false,
		},
	}
	for i, tc := range testCases {
		f := &streamFaultInjectFilter{
			config: &faultInjectConfig{
				upstream: faultUpstream,
			},
			handler: &mockStreamReceiverFilterCallbacks{
				route: &mockRoute{
					rule: tc.rule,
				},
			},
		}
		if f.matchUpstream() != tc.expected {
			t.Errorf("#%d match upstream failed", i)
		}
	}
	// upstream is empty, always returns true
	f := &streamFaultInjectFilter{
		config: &faultInjectConfig{},
	}
	if !f.matchUpstream() {
		t.Error("empty upstream not matched")
	}

}

// Delay percent should match config in errors range
func TestDelayPercent(t *testing.T) {
	percents := []uint32{1, 50, 90}
	for _, p := range percents {
		f := &streamFaultInjectFilter{
			config: &faultInjectConfig{
				delayPercent: p,
				fixedDelay:   time.Second,
			},
			rander: rand.New(rand.NewSource(time.Now().UnixNano())),
		}
		hint := uint32(0)
		testCount := uint32(1000000)
		for i := uint32(0); i < testCount; i++ {
			if f.getDelayDuration() > 0 {
				hint++
			}
		}
		//1,000,000 times, errors range in 5%
		if float32(hint)/float32(testCount)-float32(p)/100.0 > 0.05 {
			t.Errorf("percent %d's error range is not epxected, hint count %d", p, hint)
		}
	}
	nodelays := []*streamFaultInjectFilter{
		&streamFaultInjectFilter{
			config: &faultInjectConfig{
				delayPercent: 0,
				fixedDelay:   time.Second,
			},
			rander: rand.New(rand.NewSource(time.Now().UnixNano())),
		},
		&streamFaultInjectFilter{
			config: &faultInjectConfig{
				delayPercent: 100,
				fixedDelay:   0,
			},
			rander: rand.New(rand.NewSource(time.Now().UnixNano())),
		},
	}
	for _, nodelay := range nodelays {
	Run:
		for i := 0; i < 10000; i++ {
			if nodelay.getDelayDuration() > 0 {
				t.Error("nodelay get delayed")
				break Run
			}
		}
	}
	mustdelay := &streamFaultInjectFilter{
		config: &faultInjectConfig{
			delayPercent: 100,
			fixedDelay:   time.Second,
		},
		rander: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for i := 0; i < 10000; i++ {
		if mustdelay.getDelayDuration() == 0 {
			t.Error("must delay get no delay")
			break
		}
	}

}

// Abort percent should match config in errors range
func TestAbortPercent(t *testing.T) {
	percents := []uint32{1, 50, 90}
	for _, p := range percents {
		f := &streamFaultInjectFilter{
			config: &faultInjectConfig{
				abortPercent: p,
			},
			rander: rand.New(rand.NewSource(time.Now().UnixNano())),
		}
		hint := uint32(0)
		testCount := uint32(1000000)
		for i := uint32(0); i < testCount; i++ {
			if f.isAbort() {
				hint++
			}
		}
		//1,000,000 times, errors range in 5%
		if float32(hint)/float32(testCount)-float32(p)/100.0 > 0.05 {
			t.Errorf("percent %d's error range is not epxected, hint count %d", p, hint)
		}
	}
	noAbort := &streamFaultInjectFilter{
		config: &faultInjectConfig{
			abortPercent: 0,
		},
		rander: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for i := 0; i < 10000; i++ {
		if noAbort.isAbort() {
			t.Error("no abort got is abort")
			break
		}
	}
	mustAbort := &streamFaultInjectFilter{
		config: &faultInjectConfig{
			abortPercent: 100,
		},
		rander: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for i := 0; i < 10000; i++ {
		if !mustAbort.isAbort() {
			t.Error("must abort got no abort")
			break
		}
	}

}

// Test Delay and Abort Inject
func TestFaultInject_AllWithDelay(t *testing.T) {
	cfg := &v2.StreamFaultInject{
		Delay: &v2.DelayInject{
			Delay: time.Second,
			DelayInjectConfig: v2.DelayInjectConfig{
				Percent: 100,
			},
		},
		Abort: &v2.AbortInject{
			Percent: 100,
			Status:  500,
		},
	}
	cb := &mockStreamReceiverFilterCallbacks{
		info: &mockRequestInfo{},
		route: &mockRoute{
			rule: &mockRouteRule{},
		},
		called: make(chan int, 1),
	}
	f := NewFilter(context.Background(), cfg)
	f.SetReceiveFilterHandler(cb)
	start := time.Now()
	if status := f.OnReceive(context.TODO(), nil, nil, nil); status != api.StreamFilterStop {
		t.Error("fault inject should matched")
		return
	}
	select {
	case <-cb.called:
		cost := time.Now().Sub(start)
		if cost < time.Second {
			t.Error("delay not expected")
		}
		if cb.hijackCode != 500 {
			t.Error("no abort called")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout")
	}
}

func TestFaultInject_AllAbortWithoutDelay(t *testing.T) {
	cfg := &v2.StreamFaultInject{
		Abort: &v2.AbortInject{
			Percent: 100,
			Status:  500,
		},
	}
	cb := &mockStreamReceiverFilterCallbacks{
		info: &mockRequestInfo{},
		route: &mockRoute{
			rule: &mockRouteRule{},
		},
		called: make(chan int, 1),
	}
	f := NewFilter(context.Background(), cfg)
	f.SetReceiveFilterHandler(cb)
	if status := f.OnReceive(context.TODO(), nil, nil, nil); status != api.StreamFilterStop {
		t.Error("fault inject should matched")
		return
	}
	select {
	case <-cb.called:
		if cb.hijackCode != 500 {
			t.Error("no abort called")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout")
	}
}

func TestFaultInject_MatchedUpstream(t *testing.T) {
	cfg := &v2.StreamFaultInject{
		Delay: &v2.DelayInject{
			Delay: time.Second,
			DelayInjectConfig: v2.DelayInjectConfig{
				Percent: 100,
			},
		},
		UpstreamCluster: "matched",
	}
	cb := &mockStreamReceiverFilterCallbacks{
		info: &mockRequestInfo{},
		route: &mockRoute{
			rule: &mockRouteRule{
				clustername: "matched",
			},
		},
		called: make(chan int, 1),
	}
	f := NewFilter(context.Background(), cfg)
	f.SetReceiveFilterHandler(cb)
	start := time.Now()
	if status := f.OnReceive(context.TODO(), nil, nil, nil); status != api.StreamFilterContinue {
		t.Error("fault inject should matched")
		return
	}
	cost := time.Now().Sub(start)
	if cost < time.Second {
		t.Errorf("expected delay at least 1s")
	}
	notmatched := &mockStreamReceiverFilterCallbacks{
		route: &mockRoute{
			rule: &mockRouteRule{
				clustername: "notmatched",
			},
		},
	}
	f2 := NewFilter(context.Background(), cfg)
	f2.SetReceiveFilterHandler(notmatched)
	if status := f2.OnReceive(context.TODO(), nil, nil, nil); status != api.StreamFilterContinue {
		t.Error("unmatched upstream not returns continue")
	}

}

func TestFaultInject_MatchedHeader(t *testing.T) {
	cfg := &v2.StreamFaultInject{
		Delay: &v2.DelayInject{
			Delay: time.Second,
			DelayInjectConfig: v2.DelayInjectConfig{
				Percent: 100,
			},
		},
		Headers: []v2.HeaderMatcher{
			{
				Name:  "User",
				Value: "Alice",
			},
		},
	}
	cb := &mockStreamReceiverFilterCallbacks{
		info: &mockRequestInfo{},
		route: &mockRoute{
			rule: &mockRouteRule{},
		},
		called: make(chan int, 1),
	}
	f := NewFilter(context.Background(), cfg)
	f.SetReceiveFilterHandler(cb)
	headers := protocol.CommonHeader(map[string]string{
		"User": "Alice",
	})
	start := time.Now()
	if status := f.OnReceive(context.TODO(), headers, nil, nil); status != api.StreamFilterContinue {
		t.Error("fault inject should matched")
		return
	}
	cost := time.Now().Sub(start)
	if cost < time.Second {
		t.Errorf("expected delay at least 1s")
	}
	notmatched := protocol.CommonHeader(map[string]string{
		"User": "Bob",
	})
	f2 := NewFilter(context.Background(), cfg)
	f2.SetReceiveFilterHandler(cb)
	if status := f2.OnReceive(context.TODO(), notmatched, nil, nil); status != api.StreamFilterContinue {
		t.Error("unmatched headers not return continue")
	}
}

func TestFaultInject_RouteConfigOverride(t *testing.T) {
	routeConfigStr := `{
		"abort": {
			"status": 500,
			"percentage": 100
		}
	}`
	faultConfig := make(map[string]interface{})
	if err := json.Unmarshal([]byte(routeConfigStr), &faultConfig); err != nil {
		t.Fatalf("json unmarshal error, %v", err)
	}
	cfg := &v2.StreamFaultInject{}
	cb := &mockStreamReceiverFilterCallbacks{
		info: &mockRequestInfo{},
		route: &mockRoute{
			rule: &mockRouteRule{
				config: map[string]interface{}{
					v2.FaultStream: faultConfig,
				},
			},
		},
		called: make(chan int, 1),
	}
	f := NewFilter(context.Background(), cfg)
	f.SetReceiveFilterHandler(cb)
	if status := f.OnReceive(context.TODO(), nil, nil, nil); status != api.StreamFilterStop {
		t.Error("fault inject should matched")
		return
	}
	select {
	case <-cb.called:
		if cb.hijackCode != 500 {
			t.Error("no abort called")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout")
	}
}
