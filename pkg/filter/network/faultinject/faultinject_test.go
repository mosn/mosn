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
	"testing"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

func TestParseFaultInjectFilter(t *testing.T) {
	m := map[string]interface{}{
		"delay_percent":  100,
		"delay_duration": "15s",
	}
	faultInject, _ := ParseFaultInjectFilter(m)
	if !(faultInject.DelayDuration == uint64(15*time.Second) && faultInject.DelayPercent == 100) {
		t.Error("parse fault inject failed")
	}
}

type mockReadFilterCallbacks struct {
	api.ReadFilterCallbacks
	ch chan bool
}

func (m *mockReadFilterCallbacks) ContinueReading() {
	m.ch <- true
}

func TestFaultInject(t *testing.T) {
	fconfig := &v2.FaultInject{
		FaultInjectConfig: v2.FaultInjectConfig{
			DelayPercent: 100,
		},
		DelayDuration: uint64(time.Second),
	}
	injector := NewFaultInjector(fconfig)
	cb := &mockReadFilterCallbacks{
		ch: make(chan bool),
	}
	if status := injector.OnNewConnection(); status != api.Continue {
		t.Fatalf("wanna do nothing just continue, but %v", status)
	}
	injector.InitializeReadFilterCallbacks(cb)
	start := time.Now()
	if status := injector.OnData(nil); status != api.Stop {
		t.Fatalf("wanna got stop status but got %v", status)
	}
	select {
	case <-cb.ch:
		if time.Now().Sub(start) < time.Second {
			t.Fatal("inject delay not enough")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("no callback continue reading")
	}
}

type mockNetWorkFilterChainFactoryCallbacks struct {
	api.NetWorkFilterChainFactoryCallbacks
	rf api.ReadFilter
}

func (m *mockNetWorkFilterChainFactoryCallbacks) AddReadFilter(rf api.ReadFilter) {
	m.rf = rf
}

func TestFactory(t *testing.T) {
	f, _ := CreateFaultInjectFactory(map[string]interface{}{
		"delay_percent":  50,
		"delay_duration": "1s",
	})
	cb := &mockNetWorkFilterChainFactoryCallbacks{}
	f.CreateFilterChain(context.TODO(), cb)
	if cb.rf == nil {
		t.Fatal("no read filter added")
	}
}
