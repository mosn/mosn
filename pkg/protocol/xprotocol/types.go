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
	"errors"

	"mosn.io/mosn/pkg/types"
)

// StreamType distinguish the stream flow type.
// Request: stream is a normal request and needs response
// RequestOneWay: stream is a oneway request and doesn't need response
// Response: stream is a response to specific request
type StreamType int

const (
	Request StreamType = iota
	RequestOneWay
	Response
)

// Error def
var (
	AlreadyRegistered = "protocol code already registered"
	UnknownType       = "unknown model type"
	UnrecognizedCode  = "unrecognized protocol code"
	NoProtocolCode    = "no protocol code found"

	ErrDupRegistered    = errors.New(AlreadyRegistered)
	ErrUnknownType      = errors.New(UnknownType)
	ErrUnrecognizedCode = errors.New(UnrecognizedCode)
	ErrNoProtocolCode   = errors.New(NoProtocolCode)
)

// XFrame represents the minimal programmable object of the protocol.
type XFrame interface {
	// TODO: make multiplexing optional, and maybe we can support PING-PONG protocol in this framework.
	Multiplexing

	HeartbeatPredicate

	GetStreamType() StreamType

	GetHeader() types.HeaderMap

	GetData() types.IoBuffer

	SetData(data types.IoBuffer)
}

// XRespFrame expose response status code based on the XFrame
type XRespFrame interface {
	XFrame

	GetStatusCode() uint32
}

// Multiplexing provides the ability to distinguish multi-requests in single-connection by recognize 'request-id' semantics
type Multiplexing interface {
	GetRequestId() uint64

	SetRequestId(id uint64)
}

// HeartbeatPredicate provides the ability to judge if current frame is a heartbeat, which is usually used to make connection keepalive
type HeartbeatPredicate interface {
	IsHeartbeatFrame() bool
}

// ServiceAware provides the ability to get the most common info for rpc invocation: service name and method name
type ServiceAware interface {
	GetServiceName() string

	GetMethodName() string
}

// HeartbeatPredicate provides the ability to judge if current is a goaway frmae, which indicates that current connection
// should be no longer used and turn into the draining state.
type GoAwayPredicate interface {
	IsGoAwayFrame() bool
}

// XProtocol provides extra ability(Heartbeater, Hijacker) to interacts with the proxy framework based on the Protocol interface.
// e.g. A request which cannot find route should be responded with a error response like '404 Not Found', that is what Hijacker
// interface exactly provides.
type XProtocol interface {
	types.Protocol

	Heartbeater

	Hijacker
}

// HeartbeatBuilder provides the ability to construct proper heartbeat command for xprotocol sub-protocols
type Heartbeater interface {
	// Trigger builds an active heartbeat command
	Trigger(requestId uint64) XFrame

	// Reply builds heartbeat command corresponding to the given requestID
	Reply(request XFrame) XRespFrame
}

// Hijacker provides the ability to construct proper response command for xprotocol sub-protocols
type Hijacker interface {
	// BuildResponse build response with given status code
	Hijack(request XFrame, statusCode uint32) XRespFrame

	// Mapping the http status code, which used by proxy framework into protocol-specific status
	Mapping(httpStatusCode uint32) uint32
}
