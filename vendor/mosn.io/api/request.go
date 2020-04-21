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

package api

import (
	"net"
	"time"
)

type Protocol string

// ResponseFlag type
type ResponseFlag int

// Some Response Flags
const (
	// no healthy upstream found
	NoHealthyUpstream ResponseFlag = 0x2
	// Upstream Request timeout
	UpstreamRequestTimeout ResponseFlag = 0x4
	// local reset
	UpstreamLocalReset ResponseFlag = 0x8
	// upstream reset
	UpstreamRemoteReset ResponseFlag = 0x10
	// connect upstream failure
	UpstreamConnectionFailure ResponseFlag = 0x20
	// upstream terminate connection
	UpstreamConnectionTermination ResponseFlag = 0x40
	// upstream's connection overflow
	UpstreamOverflow ResponseFlag = 0x80
	// no route found
	NoRouteFound ResponseFlag = 0x100
	// inject delay
	DelayInjected ResponseFlag = 0x200
	// inject fault
	FaultInjected ResponseFlag = 0x400
	// rate limited
	RateLimited ResponseFlag = 0x800
	// payload limit
	ReqEntityTooLarge ResponseFlag = 0x1000
)

// RequestInfo has information for a request, include the basic information,
// the request's downstream information, ,the request's upstream information and the router information.
type RequestInfo interface {
	// StartTime returns the time that request arriving
	StartTime() time.Time

	// SetStartTime sets StartTime
	SetStartTime()

	// RequestReceivedDuration returns duration between request arriving and request forwarding to upstream
	RequestReceivedDuration() time.Duration

	// SetRequestReceivedDuration sets duration between request arriving and request forwarding to upstream
	SetRequestReceivedDuration(time time.Time)

	// ResponseReceivedDuration gets duration between request arriving and response received
	ResponseReceivedDuration() time.Duration

	// SetResponseReceivedDuration sets duration between request arriving and response received
	SetResponseReceivedDuration(time time.Time)

	// RequestFinishedDuration returns duration between request arriving and request finished
	RequestFinishedDuration() time.Duration

	// SetRequestFinishedDuration sets duration between request arriving and request finished
	SetRequestFinishedDuration(time time.Time)

	// SetProcessTimeDuration sets duration in mosn
	SetProcessTimeDuration(d time.Duration)

	// ProcessTimeDuration returns duration between mosn
	ProcessTimeDuration() time.Duration

	// BytesSent reports the bytes sent
	BytesSent() uint64

	// SetBytesSent sets the bytes sent
	SetBytesSent(bytesSent uint64)

	// BytesReceived reports the bytes received
	BytesReceived() uint64

	// SetBytesReceived sets the bytes received
	SetBytesReceived(bytesReceived uint64)

	// Protocol returns the request's protocol type
	Protocol() Protocol
	// SetProtocol sets the request's protocol type
	SetProtocol(p Protocol)

	// ResponseCode reports the request's response code
	// The code is http standard status code.
	ResponseCode() int

	// SetResponseCode set request's response code
	// Mosn use http standard status code for log, if a protocol have different status code
	// we will try to mapping it to http status code, and log it
	SetResponseCode(code int)

	// Duration reports the duration since request's starting time
	Duration() time.Duration

	// GetResponseFlag gets request's response flag
	GetResponseFlag(flag ResponseFlag) bool

	// SetResponseFlag sets request's response flag
	SetResponseFlag(flag ResponseFlag)

	//UpstreamHost reports  the selected upstream's host information
	UpstreamHost() HostInfo

	// OnUpstreamHostSelected sets the selected upstream's host information
	OnUpstreamHostSelected(host HostInfo)

	// UpstreamLocalAddress reports the upstream's local network address
	UpstreamLocalAddress() string

	// SetUpstreamLocalAddress sets upstream's local network address
	SetUpstreamLocalAddress(localAddress string)

	// IsHealthCheck checks whether the request is health.
	IsHealthCheck() bool

	// SetHealthCheck sets the request's health state.
	SetHealthCheck(isHc bool)

	// DownstreamLocalAddress reports the downstream's local network address.
	DownstreamLocalAddress() net.Addr

	// SetDownstreamLocalAddress sets the downstream's local network address.
	SetDownstreamLocalAddress(addr net.Addr)

	// DownstreamRemoteAddress reports the downstream's remote network address.
	DownstreamRemoteAddress() net.Addr

	// SetDownstreamRemoteAddress sets the downstream's remote network address.
	SetDownstreamRemoteAddress(addr net.Addr)

	// RouteEntry reports the route rule
	RouteEntry() RouteRule

	// SetRouteEntry sets the route rule
	SetRouteEntry(routerRule RouteRule)
}
