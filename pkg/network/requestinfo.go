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

package network

import (
	"net"
	"time"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// RequestInfo
type RequestInfo struct {
	protocol                 types.Protocol
	startTime                time.Time
	responseFlag             types.ResponseFlag
	upstreamHost             types.HostInfo
	requestReceivedDuration  time.Duration
	responseReceivedDuration time.Duration
	bytesSent                uint64
	bytesReceived            uint64
	responseCode             uint32
	localAddress             net.Addr
	downstreamLocalAddress   net.Addr
	downstreamRemoteAddress  net.Addr
	isHealthCheckRequest     bool
	routerRule               types.RouteRule
}

// todo check
func newRequestInfoWithPort(protocol types.Protocol) types.RequestInfo {
	return &RequestInfo{
		protocol:  protocol,
		startTime: time.Now(),
	}
}

// NewrequestInfo
func NewRequestInfo() types.RequestInfo {
	return &RequestInfo{
		startTime: time.Now(),
	}
}

func (r *RequestInfo) StartTime() time.Time {
	return r.startTime
}

func (r *RequestInfo) SetStartTime() {
    r.startTime = time.Now()
}

func (r *RequestInfo) RequestReceivedDuration() time.Duration {
	return r.requestReceivedDuration
}

func (r *RequestInfo) SetRequestReceivedDuration(time time.Time) {
	r.requestReceivedDuration = time.Sub(r.startTime)
}

func (r *RequestInfo) ResponseReceivedDuration() time.Duration {
	return r.responseReceivedDuration
}

func (r *RequestInfo) SetResponseReceivedDuration(time time.Time) {
	r.responseReceivedDuration = time.Sub(r.startTime)
}

func (r *RequestInfo) BytesSent() uint64 {
	return r.bytesSent
}

func (r *RequestInfo) SetBytesSent(bytesSent uint64) {
	r.bytesSent = bytesSent
}

func (r *RequestInfo) BytesReceived() uint64 {
	return r.bytesReceived
}

func (r *RequestInfo) SetBytesReceived(bytesReceived uint64) {
	r.bytesReceived = bytesReceived
}

func (r *RequestInfo) Protocol() types.Protocol {
	return r.protocol
}

func (r *RequestInfo) ResponseCode() uint32 {
	return r.responseCode
}

func (r *RequestInfo) Duration() time.Duration {
	return time.Now().Sub(r.startTime)
}

func (r *RequestInfo) GetResponseFlag(flag types.ResponseFlag) bool {
	return r.responseFlag&flag != 0
}

func (r *RequestInfo) SetResponseFlag(flag types.ResponseFlag) {
	r.responseFlag |= flag
}

func (r *RequestInfo) UpstreamHost() types.HostInfo {
	return r.upstreamHost
}

func (r *RequestInfo) OnUpstreamHostSelected(host types.HostInfo) {
	r.upstreamHost = host
}

func (r *RequestInfo) UpstreamLocalAddress() net.Addr {
	return r.localAddress
}

func (r *RequestInfo) SetUpstreamLocalAddress(addr net.Addr) {
	r.localAddress = addr
}

func (r *RequestInfo) IsHealthCheck() bool {
	return r.isHealthCheckRequest
}

func (r *RequestInfo) SetHealthCheck(isHc bool) {
	r.isHealthCheckRequest = isHc
}

func (r *RequestInfo) DownstreamLocalAddress() net.Addr {
	return r.downstreamLocalAddress
}

func (r *RequestInfo) SetDownstreamLocalAddress(addr net.Addr) {
	r.downstreamLocalAddress = addr
}

func (r *RequestInfo) DownstreamRemoteAddress() net.Addr {
	return r.downstreamRemoteAddress
}

func (r *RequestInfo) SetDownstreamRemoteAddress(addr net.Addr) {
	r.downstreamRemoteAddress = addr
}

func (r *RequestInfo) RouteEntry() types.RouteRule {
	return r.routerRule
}

func (r *RequestInfo) SetRouteEntry(routerRule types.RouteRule) {
	r.routerRule = routerRule
}
