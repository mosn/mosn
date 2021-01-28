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

package xprotocol

import (
	"mosn.io/api/protocol/xprotocol"
)

// StreamType distinguish the stream flow type.
// Request: stream is a normal request and needs response
// RequestOneWay: stream is a oneway request and doesn't need response
// Response: stream is a response to specific request
// Deprecated: use mosn.io/api/protocol/xprotocol/types.go:StreamType instead
type StreamType = xprotocol.StreamType

const (
	Request       = xprotocol.Request
	RequestOneWay = xprotocol.RequestOneWay
	Response      = xprotocol.Response
)

// Error def
// Deprecated
var (
	AlreadyRegistered = xprotocol.AlreadyRegistered
	UnknownType       = xprotocol.UnknownType
	UnrecognizedCode  = xprotocol.UnrecognizedCode
	NoProtocolCode    = xprotocol.NoProtocolCode

	ErrDupRegistered    = xprotocol.ErrDupRegistered
	ErrUnknownType      = xprotocol.ErrUnknownType
	ErrUnrecognizedCode = xprotocol.ErrUnrecognizedCode
	ErrNoProtocolCode   = xprotocol.ErrNoProtocolCode
)

// XFrame represents the minimal programmable object of the protocol.
// Deprecated: use mosn.io/api/protocol/xprotocol/types.go:XFrame instead
type XFrame = xprotocol.XFrame

// XRespFrame expose response status code based on the XFrame
// Deprecated: use mosn.io/api/protocol/xprotocol/types.go:XRespFrame instead
type XRespFrame = xprotocol.XRespFrame

// Multiplexing provides the ability to distinguish multi-requests in single-connection by recognize 'request-id' semantics
// Deprecated: use mosn.io/api/protocol/xprotocol/types.go:Multiplexing instead
type Multiplexing = xprotocol.Multiplexing

// HeartbeatPredicate provides the ability to judge if current frame is a heartbeat, which is usually used to make connection keepalive
// Deprecated: use mosn.io/api/protocol/xprotocol/types.go:HeartbeatPredicate instead
type HeartbeatPredicate = xprotocol.HeartbeatPredicate

// ServiceAware provides the ability to get the most common info for rpc invocation: service name and method name
// Deprecated: use mosn.io/api/protocol/xprotocol/types.go:ServiceAware instead
type ServiceAware = xprotocol.ServiceAware

// HeartbeatPredicate provides the ability to judge if current is a goaway frmae, which indicates that current connection
// should be no longer used and turn into the draining state.
// Deprecated: use mosn.io/api/protocol/xprotocol/types.go:GoAwayPredicate instead
type GoAwayPredicate = xprotocol.GoAwayPredicate

// XProtocol provides extra ability(Heartbeater, Hijacker) to interacts with the proxy framework based on the Protocol interface.
// e.g. A request which cannot find route should be responded with a error response like '404 Not Found', that is what Hijacker
// interface exactly provides.
// Deprecated: use mosn.io/api/protocol/xprotocol/types.go:XProtocol instead
type XProtocol = xprotocol.XProtocol

// HeartbeatBuilder provides the ability to construct proper heartbeat command for xprotocol sub-protocols
// Deprecated: use mosn.io/api/protocol/xprotocol/types.go:Heartbeater instead
type Heartbeater = xprotocol.Heartbeater

// Hijacker provides the ability to construct proper response command for xprotocol sub-protocols
// Deprecated: use mosn.io/api/protocol/xprotocol/types.go:Hijacker instead
type Hijacker = xprotocol.Hijacker
