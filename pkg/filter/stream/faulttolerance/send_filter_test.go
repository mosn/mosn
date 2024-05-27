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

package faulttolerance

import (
	"context"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/regulator"

	"mosn.io/api"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/types"
)

type mockRouteRule struct {
	api.RouteRule
}

func (r *mockRouteRule) ClusterName(context.Context) string {
	return "mockCluster"
}

type mockSendHandler struct {
	api.StreamSenderFilterHandler
	requestInfo types.RequestInfo
}

func (m *mockSendHandler) RequestInfo() types.RequestInfo {
	return m.requestInfo
}

func (m *mockSendHandler) setRequestInfo(info types.RequestInfo) {
	m.requestInfo = info
}

type mockHostInfo struct {
	api.HostInfo
	addr        string
	healthFlags uint64
}

func (m *mockHostInfo) AddressString() string {
	return m.addr
}

func (m *mockHostInfo) SetAddressString(addr string) {
	m.addr = addr
}

func (m *mockHostInfo) ClearHealthFlag(flag api.HealthFlag) {
	m.healthFlags &= ^uint64(flag)
}

func (m *mockHostInfo) ContainHealthFlag(flag api.HealthFlag) bool {
	return m.healthFlags&uint64(flag) > 0
}

func (m *mockHostInfo) SetHealthFlag(flag api.HealthFlag) {
	m.healthFlags |= uint64(flag)
}

func (m *mockHostInfo) HealthFlag() api.HealthFlag {
	return api.HealthFlag(m.healthFlags)
}

func (m *mockHostInfo) Health() bool {
	return m.healthFlags == 0
}

func TestAppendFilter(t *testing.T) {

	cfg := &v2.FaultToleranceFilterConfig{
		Enabled:               true,
		TimeWindow:            1000,
		MaxIpCount:            1,
		LeastWindowCount:      1,
		ExceptionTypes:        map[uint32]bool{502: true, 504: true},
		RecoverTime:           3000,
		TaskSize:              10,
		ExceptionRateMultiple: 1,
	}

	f := NewSendFilter(cfg, regulator.NewInvocationStatFactory(cfg))

	addr1 := "127.0.0.1:80"
	addr2 := "127.0.0.1:81"
	host1 := &mockHostInfo{addr: addr1}
	host2 := &mockHostInfo{addr: addr2}
	sendHandler := &mockSendHandler{}

	info1 := &network.RequestInfo{}
	info1.OnUpstreamHostSelected(host1)
	info1.SetRouteEntry(&mockRouteRule{})
	info1.SetResponseCode(504)

	info2 := &network.RequestInfo{}
	info2.OnUpstreamHostSelected(host2)
	info2.SetRouteEntry(&mockRouteRule{})
	info2.SetResponseCode(200)

	f.SetSenderFilterHandler(sendHandler)

	sendHandler.setRequestInfo(info1)
	f.Append(context.Background(), nil, nil, nil)

	sendHandler.setRequestInfo(info2)
	f.Append(context.Background(), nil, nil, nil)

	sendHandler.setRequestInfo(info1)
	f.Append(context.Background(), nil, nil, nil)

	// host1 should set unhealth
	// The maximum delay time is 2s (500ms(task check interval) + 1s(TimeWindow) + 500ms(task check interval))
	time.Sleep(time.Duration(cfg.TimeWindow/1000+2) * time.Second)
	if host1.Health() {
		t.Errorf("health status should false, but got health status: %v", host1.Health())
	}

	// host2 not be set unhealth
	if !host2.Health() {
		t.Errorf("health status should true, but got health status: %v", host2.Health())
	}

	// After cfg.RecoverTime, the host1 health should be recover
	time.Sleep(time.Duration(cfg.RecoverTime/1000+1) * time.Second)
	if !host1.Health() {
		t.Errorf("health status should recover,but got health status: %v", host1.Health())
	}

	// test IsException
	if isException := f.IsException(info1); !isException {
		t.Error("info1 should be isException")
	}
	if isException := f.IsException(info2); isException {
		t.Error("info2 should not be isException")
	}
}

func TestCreateSendFilterFactory(t *testing.T) {
	m := map[string]interface{}{}
	if f, err := CreateSendFilterFactory(m); err != nil || f == nil {
		t.Errorf("CreateSendFilterFactory failed error: %v", err)
	}

	m["enabled"] = true
	m["exceptionTypes"] = []uint32{502, 503}
	if f, err := CreateSendFilterFactory(m); err != nil || f == nil {
		t.Errorf("CreateSendFilterFactory failed error: %v", err)
	}

}
