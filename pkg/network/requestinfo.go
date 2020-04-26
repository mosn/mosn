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

	"mosn.io/api"
)

// RequestInfo
type RequestInfo struct {
	protocol                 api.Protocol
	startTime                time.Time
	responseFlag             api.ResponseFlag
	upstreamHost             api.HostInfo
	requestReceivedDuration  time.Duration
	requestFinishedDuration  time.Duration
	responseReceivedDuration time.Duration
	processTimeDuration      time.Duration
	bytesSent                uint64
	bytesReceived            uint64
	responseCode             int
	localAddress             string
	downstreamLocalAddress   net.Addr
	downstreamRemoteAddress  net.Addr
	isHealthCheckRequest     bool
	routerRule               api.RouteRule
}

func newRequestInfoWithPort(protocol api.Protocol) api.RequestInfo {
	return &RequestInfo{
		protocol:  protocol,
		startTime: time.Now(),
	}
}

// NewrequestInfo
func NewRequestInfo() api.RequestInfo {
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

func (r *RequestInfo) SetRequestReceivedDuration(t time.Time) {
	r.requestReceivedDuration = t.Sub(r.startTime)
}

func (r *RequestInfo) ResponseReceivedDuration() time.Duration {
	return r.responseReceivedDuration
}

func (r *RequestInfo) SetResponseReceivedDuration(t time.Time) {
	r.responseReceivedDuration = t.Sub(r.startTime)
}

func (r *RequestInfo) RequestFinishedDuration() time.Duration {
	return r.requestFinishedDuration
}

func (r *RequestInfo) SetRequestFinishedDuration(t time.Time) {
	r.requestFinishedDuration = t.Sub(r.startTime)

}

func (r *RequestInfo) ProcessTimeDuration() time.Duration {
	return r.processTimeDuration
}

func (r *RequestInfo) SetProcessTimeDuration(d time.Duration) {
	r.processTimeDuration = d
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

func (r *RequestInfo) Protocol() api.Protocol {
	return r.protocol
}

func (r *RequestInfo) SetProtocol(p api.Protocol) {
	r.protocol = p
}

func (r *RequestInfo) ResponseCode() int {
	return r.responseCode
}

func (r *RequestInfo) SetResponseCode(code int) {
	r.responseCode = code
}

func (r *RequestInfo) Duration() time.Duration {
	return time.Now().Sub(r.startTime)
}

func (r *RequestInfo) GetResponseFlag(flag api.ResponseFlag) bool {
	return r.responseFlag&flag != 0
}

func (r *RequestInfo) SetResponseFlag(flag api.ResponseFlag) {
	r.responseFlag |= flag
}

func (r *RequestInfo) UpstreamHost() api.HostInfo {
	return r.upstreamHost
}

func (r *RequestInfo) OnUpstreamHostSelected(host api.HostInfo) {
	r.upstreamHost = host
}

func (r *RequestInfo) UpstreamLocalAddress() string {
	return r.localAddress
}

func (r *RequestInfo) SetUpstreamLocalAddress(addr string) {
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

func (r *RequestInfo) RouteEntry() api.RouteRule {
	return r.routerRule
}

func (r *RequestInfo) SetRouteEntry(routerRule api.RouteRule) {
	r.routerRule = routerRule
}
