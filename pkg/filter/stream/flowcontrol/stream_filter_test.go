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

package flowcontrol

import (
	"context"
	"net"
	"testing"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

const (
	HTTP1 types.ProtocolName = "Http1"
	Dubbo types.ProtocolName = "Dubbo"
)

func MockInboundFilter(mockConfig *Config) *StreamFilter {
	sf := NewStreamFilter(&DefaultCallbacks{config: mockConfig}, base.Inbound)
	sf.SetReceiveFilterHandler(&mockStreamReceiverFilterHandler{})
	sf.SetSenderFilterHandler(&mockStreamSenderFilterHandler{})
	return sf
}
func TestStreamFilter(t *testing.T) {

	mockConfig := &Config{
		GlobalSwitch: false,
		Monitor:      false,
		KeyType:      "PATH",
		Rules: []*flow.Rule{
			{
				ID:              "0",
				Resource:        "/http",
				Threshold:       1,
				ControlBehavior: 0,
			},
		},
	}
	flow.LoadRules(mockConfig.Rules)
	// global switch disabled
	sf := MockInboundFilter(mockConfig)

	status := sf.OnReceive(context.Background(), nil, nil, nil)
	assert.Equal(t, api.StreamFilterContinue, status)

	// global switch enabled
	mockConfig.GlobalSwitch = true
	status = sf.OnReceive(context.Background(), nil, nil, nil)
	assert.Equal(t, api.StreamFilterContinue, status)

	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, HTTP1)

	m := make(map[string]string)
	m["Http1_request_path"] = "/http"
	for k := range m {
		// register test variable
		variable.Register(variable.NewStringVariable(k, nil,
			func(ctx context.Context, variableValue *variable.IndexedValue, data interface{}) (s string, err error) {
				val := m[k]
				return val, nil
			}, nil, 0))
	}
	variable.RegisterProtocolResource(HTTP1, api.PATH, "request_path")

	// test pass
	status = sf.OnReceive(ctx, nil, nil, nil)
	assert.Equal(t, api.StreamFilterContinue, status)
	// test block
	for i := 0; i < 5; i++ {
		status = sf.OnReceive(ctx, nil, nil, nil)
	}
	assert.Equal(t, api.StreamFilterStop, status)
	sf.OnDestroy()
}

func BenchmarkStreamFilter_OnReceive(b *testing.B) {
	mockConfig := &Config{
		GlobalSwitch: true,
		Monitor:      false,
		KeyType:      "PATH",
	}
	cb := &DefaultCallbacks{
		config: mockConfig,
	}
	filter := NewStreamFilter(cb, base.Inbound)

	filter.SetReceiveFilterHandler(&mockStreamReceiverFilterHandler{})
	filter.SetSenderFilterHandler(&mockStreamSenderFilterHandler{})

	header := mockRPCHeader("testingService", "sum")
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.OnReceive(ctx, header, nil, nil)
	}
}

func BenchmarkStreamFilter_OnReceive_SwitchOn(b *testing.B) {
	mockConfig := &Config{
		GlobalSwitch: true,
		Monitor:      false,
		KeyType:      "PATH",
	}
	cb := &DefaultCallbacks{
		config: mockConfig,
	}
	filter := NewStreamFilter(cb, base.Inbound)

	filter.SetReceiveFilterHandler(&mockStreamReceiverFilterHandler{})
	filter.SetSenderFilterHandler(&mockStreamSenderFilterHandler{})

	header := mockRPCHeader("testingService", "sum")
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.OnReceive(ctx, header, nil, nil)
	}
}

type mockStreamReceiverFilterHandler struct {
	api.StreamReceiverFilterHandler
}

func (m *mockStreamReceiverFilterHandler) RequestInfo() api.RequestInfo {
	mockReqInfo := network.NewRequestInfo()
	mockReqInfo.OnUpstreamHostSelected(&MockHost{
		ip: "127.0.0.1",
	})
	ri := &mockRequestInfo{
		RequestInfo: mockReqInfo,
	}
	return ri
}

func (m *mockStreamReceiverFilterHandler) SendDirectResponse(headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) {

}

func (m *mockStreamReceiverFilterHandler) Connection() api.Connection {
	mc := &mockConnection{}
	return mc
}

type mockStreamSenderFilterHandler struct {
	api.StreamSenderFilterHandler
}

func (m *mockStreamSenderFilterHandler) RequestInfo() api.RequestInfo {
	ri := &mockRequestInfo{
		flags: map[api.ResponseFlag]bool{
			api.RateLimited: true,
		},
	}
	return ri
}

func (m *mockStreamSenderFilterHandler) Connection() api.Connection {
	return nil
}

type mockRequestInfo struct {
	api.RequestInfo
	flags map[api.ResponseFlag]bool
	code  int
}

func (m *mockRequestInfo) GetResponseFlag(flag api.ResponseFlag) bool {
	return m.flags[flag]
}

func (m *mockRequestInfo) ResponseCode() int {
	return m.code
}

func mockRPCHeader(service string, method string) types.HeaderMap {
	mockHeader := &mockHeaderMap{
		headers: map[string]string{},
	}
	// mockHeader.Set(constants.SOFARPC_ROUTER_HEADER_TARGET_SERVICE_UNIQUE_NAME, service)
	// mockHeader.Set(constants.SOFARPC_ROUTER_HEADER_METHOD_NAME, method)
	return mockHeader
}

type MockHost struct {
	ip          string
	healthFlags uint64
}

func (m *MockHost) CreateConnection(context context.Context) types.CreateConnectionData {
	return types.CreateConnectionData{}
}

func (m *MockHost) ClearHealthFlag(flag api.HealthFlag) {
	m.healthFlags &= ^uint64(flag)
}

func (m *MockHost) ContainHealthFlag(flag api.HealthFlag) bool {
	return m.healthFlags&uint64(flag) > 0
}

func (m *MockHost) SetHealthFlag(flag api.HealthFlag) {
	m.healthFlags |= uint64(flag)
}

func (m *MockHost) HealthFlag() api.HealthFlag {
	return api.HealthFlag(m.healthFlags)
}

func (m *MockHost) Health() bool {
	return m.healthFlags == 0
}

func (m *MockHost) Hostname() string {
	return ""
}

// Metadata returns the host's meta data
func (m *MockHost) Metadata() api.Metadata {
	return nil
}

// ClusterInfo returns the cluster info
func (m *MockHost) ClusterInfo() types.ClusterInfo {
	return nil
}

// Address returns the host's Addr structure
func (m *MockHost) Address() net.Addr {
	return nil
}

// AddressString retuens the host's address string
func (m *MockHost) AddressString() string {
	return m.ip
}

// HostStats returns the host stats metrics
func (m *MockHost) HostStats() types.HostStats {
	return types.HostStats{}
}

// Weight returns the host weight
func (m *MockHost) Weight() uint32 {
	return 0
}

// Config creates a host config by the host attributes
func (m *MockHost) Config() v2.Host {
	return v2.Host{}
}

func (m *MockHost) SupportTLS() bool {
	return false
}

type MockSendHandler struct {
}

func (m MockSendHandler) Route() api.Route {
	panic("implement me")
}

var (
	mockErrorRI = &mockRequestInfo{
		flags: map[api.ResponseFlag]bool{
			api.UpstreamRequestTimeout: true,
		},
	}
	mockNormalRI = &mockRequestInfo{
		flags: map[api.ResponseFlag]bool{},
		code:  api.SuccessCode,
	}
)

func (m MockSendHandler) RequestInfo() api.RequestInfo {
	return mockNormalRI
}

func (m MockSendHandler) Connection() api.Connection {
	panic("implement me")
}

func (m MockSendHandler) GetResponseHeaders() api.HeaderMap {
	panic("implement me")
}

func (m MockSendHandler) SetResponseHeaders(headers api.HeaderMap) {
	panic("implement me")
}

func (m MockSendHandler) GetResponseData() buffer.IoBuffer {
	panic("implement me")
}

func (m MockSendHandler) SetResponseData(buf buffer.IoBuffer) {
	panic("implement me")
}

func (m MockSendHandler) GetResponseTrailers() api.HeaderMap {
	panic("implement me")
}

func (m MockSendHandler) SetResponseTrailers(trailers api.HeaderMap) {
	panic("implement me")
}

type mockConnection struct {
	api.Connection
}

func (m mockConnection) RemoteAddr() net.Addr {
	ma := &mockAddr{}
	return ma
}

type mockAddr struct {
	net.Addr
}

func (m mockAddr) Network() string {
	return ""
}
func (m mockAddr) String() string {
	return ""
}

type mockHeaderMap struct {
	api.HeaderMap
	headers map[string]string
}

func (headers *mockHeaderMap) Get(key string) (string, bool) {
	if value, ok := headers.headers[key]; ok {
		return value, true
	} else {
		return "", false
	}
}

func (h *mockHeaderMap) Set(key, value string) {
	h.headers[key] = value
}

func (h *mockHeaderMap) Del(key string) {
	delete(h.headers, key)
}

func TestStreamFilter_Append(t *testing.T) {
	cb := &DefaultCallbacks{}
	filter := NewStreamFilter(cb, base.Inbound)

	filter.SetReceiveFilterHandler(&mockStreamReceiverFilterHandler{})
	filter.SetSenderFilterHandler(&mockStreamSenderFilterHandler{})

	header := mockRPCHeader("testingService", "sum")
	ctx := context.Background()

	filter.BlockError = base.NewBlockError(base.WithBlockType(base.BlockTypeFlow))
	assert.Equal(t, api.StreamFilterStop, filter.Append(ctx, header, nil, nil))

	// clear block error, nil block error and Entry
	filter.BlockError = nil
	assert.Equal(t, api.StreamFilterContinue, filter.Append(ctx, header, nil, nil))

	// set entry
	entry, err := sentinel.Entry("test")
	assert.Nil(t, err)
	filter.Entry = entry
	assert.Equal(t, api.StreamFilterContinue, filter.Append(ctx, header, nil, nil))

}
