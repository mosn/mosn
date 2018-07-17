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
type requestInfo struct {
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

func NewRequestInfoWithPort(protocol types.Protocol) types.RequestInfo {
	return &requestInfo{
		protocol:  protocol,
		startTime: time.Now(),
	}
}

func NewRequestInfo() types.RequestInfo {
	return &requestInfo{
		startTime: time.Now(),
	}
}

func (r *requestInfo) StartTime() time.Time {
	return r.startTime
}

func (r *requestInfo) RequestReceivedDuration() time.Duration {
	return r.requestReceivedDuration
}

func (r *requestInfo) SetRequestReceivedDuration(time time.Time) {
	r.requestReceivedDuration = time.Sub(r.startTime)
}

func (r *requestInfo) ResponseReceivedDuration() time.Duration {
	return r.responseReceivedDuration
}

func (r *requestInfo) SetResponseReceivedDuration(time time.Time) {
	r.responseReceivedDuration = time.Sub(r.startTime)
}

func (r *requestInfo) BytesSent() uint64 {
	return r.bytesSent
}

func (r *requestInfo) SetBytesSent(bytesSent uint64) {
	r.bytesSent = bytesSent
}

func (r *requestInfo) BytesReceived() uint64 {
	return r.bytesReceived
}

func (r *requestInfo) SetBytesReceived(bytesReceived uint64) {
	r.bytesReceived = bytesReceived
}

func (r *requestInfo) Protocol() types.Protocol {
	return r.protocol
}

func (r *requestInfo) ResponseCode() uint32 {
	return r.responseCode
}

func (r *requestInfo) Duration() time.Duration {
	return time.Now().Sub(r.startTime)
}

func (r *requestInfo) GetResponseFlag(flag types.ResponseFlag) bool {
	return r.responseFlag&flag != 0
}

func (r *requestInfo) SetResponseFlag(flag types.ResponseFlag) {
	r.responseFlag |= flag
}

func (r *requestInfo) UpstreamHost() types.HostInfo {
	return r.upstreamHost
}

func (r *requestInfo) OnUpstreamHostSelected(host types.HostInfo) {
	r.upstreamHost = host
}

func (r *requestInfo) UpstreamLocalAddress() net.Addr {
	return r.localAddress
}

func (r *requestInfo) SetUpstreamLocalAddress(addr net.Addr) {
	r.localAddress = addr
}

func (r *requestInfo) IsHealthCheck() bool {
	return r.isHealthCheckRequest
}

func (r *requestInfo) SetHealthCheck(isHc bool) {
	r.isHealthCheckRequest = isHc
}

func (r *requestInfo) DownstreamLocalAddress() net.Addr {
	return r.downstreamLocalAddress
}

func (r *requestInfo) SetDownstreamLocalAddress(addr net.Addr) {
	r.downstreamLocalAddress = addr
}

func (r *requestInfo) DownstreamRemoteAddress() net.Addr {
	return r.downstreamRemoteAddress
}

func (r *requestInfo) SetDownstreamRemoteAddress(addr net.Addr) {
	r.downstreamRemoteAddress = addr
}

func (r *requestInfo) RouteEntry() types.RouteRule {
	return r.routerRule
}

func (r *requestInfo) SetRouteEntry(routerRule types.RouteRule) {
	r.routerRule = routerRule
}
