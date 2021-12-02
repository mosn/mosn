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

package ipaccess

import (
	"context"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"net"
	"testing"
)

func TestCreateIPAccessFactory(t *testing.T) {
	m := map[string]interface{}{
		"header": "x-forward-ip",
		"ips":    []*IpAndAction{{Action: allow, Addrs: []string{"127.0.0.1"}}},
	}
	if f, err := CreateIPAccessFactory(m); f == nil || err != nil {
		t.Errorf("CreateIPAccessFactory failed: %v.", err)
	}

}

func TestIPAccessFilter_OnReceive(t *testing.T) {
	ipList, _ := NewIpList([]string{"127.0.0.1", "192.168.0.1/24", "127.0.2.1", "172.12.1.1/16"})
	ip192, _ := NewIpList([]string{"192.168.1.1"})
	net24, _ := NewIpList([]string{"192.168.1.0/24"})
	net16, _ := NewIpList([]string{"10.1.1.0/16"})
	netv6, _ := NewIpList([]string{"2001:0db8:0000:0000:0000:0000:0000:0000/32"})
	all, _ := NewIpList([]string{"0.0.0.0/0"})
	ips := []IPAccess{&IPBlocklist{ip192}, &IPAllowlist{net24}, &IPAllowlist{net16}, &IPAllowlist{netv6}, &IPBlocklist{all}}

	data := []struct {
		ipList   []IPAccess
		ip       string
		allowAll bool
		exp      bool
	}{
		{[]IPAccess{&IPAllowlist{ipList}}, "127.0.0.1", true, true},
		{[]IPAccess{&IPAllowlist{ipList}, &IPBlocklist{all}}, "127.0.0.2", true, false},
		{[]IPAccess{&IPBlocklist{ipList}}, "127.0.0.1", true, false},
		{[]IPAccess{&IPBlocklist{ipList}}, "127.0.0.2", true, true},
		{[]IPAccess{&IPAllowlist{ip192}, &IPBlocklist{net24}, &IPAllowlist{all}}, "192.168.1.1", true, true},
		{[]IPAccess{&IPAllowlist{ip192}, &IPBlocklist{net24}, &IPAllowlist{all}}, "127.0.0.2", true, true},
		{[]IPAccess{&IPAllowlist{ip192}, &IPBlocklist{net24}, &IPAllowlist{all}}, "192.168.1.12", true, false},
		{ips, "192.168.1.1", true, false},
		{ips, "192.168.1.2", true, true},
		{ips, "2001:0db8:0000:0000:0000:0000:0000:0000", true, true},
		{ips, "172.168.1.2", true, false},
		{[]IPAccess{}, "123", true, true},
		{[]IPAccess{}, "123", false, false},
		{[]IPAccess{&IPAllowlist{ip192}}, "123", false, false},
	}

	ctx := context.Background()

	for i, v := range data {
		header := headerMap("x-forward-ip", v.ip)
		m := &mockStreamReceiverFilterHandler{}
		filter := NewIPAccessFilter(v.ipList, "x-forward-ip", v.allowAll)
		filter.SetReceiveFilterHandler(m)
		receive := filter.OnReceive(ctx, header, nil, nil)
		result := receive == api.StreamFilterContinue
		if result != v.exp {
			t.Errorf("test failed ip:%s,index: %d,", v.ip, i)
		}
	}
}

func BenchmarkIPAccessFilter_OnReceive(b *testing.B) {
	allowlist, err := NewIpList([]string{"127.0.0.1", "192.168.0.1/24", "127.0.2.1", "172.12.1.1/16"})
	if err != nil {
		b.Errorf("failed build ipaccess ：%v", err)
	}
	filter := NewIPAccessFilter([]IPAccess{&IPAllowlist{allowlist}}, "x-forward-ip", true)
	filter.SetReceiveFilterHandler(&mockStreamReceiverFilterHandler{})
	header := headerMap("x-forward-ip", "127.0.0.1")
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.OnReceive(ctx, header, nil, nil)
	}
}

type mockSendHandler struct {
	api.StreamSenderFilterHandler
	downstreamRespDataBuf types.IoBuffer
}

func (f *mockSendHandler) SetResponseData(data types.IoBuffer) {
	// data is the original data. do nothing
	if f.downstreamRespDataBuf == data {
		return
	}
	if f.downstreamRespDataBuf == nil {
		f.downstreamRespDataBuf = buffer.NewIoBuffer(0)
	}
	f.downstreamRespDataBuf.Reset()
	f.downstreamRespDataBuf.ReadFrom(data)
}

func (f *mockSendHandler) setData(data types.IoBuffer) {
	f.downstreamRespDataBuf = data
}

type mockStreamReceiverFilterHandler struct {
	api.StreamReceiverFilterHandler
}

func (m *mockStreamReceiverFilterHandler) SendHijackReply(code int, headerMap types.HeaderMap) {

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
func (m *mockRequestInfo) Protocol() api.ProtocolName {
	return protocol.HTTP1
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

func headerMap(key string, val string) types.HeaderMap {
	mockHeader := &mockHeaderMap{
		headers: map[string]string{key: val},
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

type mockConnection struct {
	api.Connection
}

func (m mockConnection) RemoteAddr() net.Addr {

	return &net.IPNet{
		IP: net.ParseIP("127.0.0.1"),
	}
}

type mockAddr struct {
	addr net.Addr
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
