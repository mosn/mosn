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
	"context"
	"errors"
)

// status codes.
const (
	CodecExceptionCode    = 0
	UnknownCode           = 2
	DeserialExceptionCode = 3
	SuccessCode           = 200
	PermissionDeniedCode  = 403
	RouterUnavailableCode = 404
	InternalErrorCode     = 500
	NoHealthUpstreamCode  = 502
	UpstreamOverFlowCode  = 503
	TimeoutExceptionCode  = 504
	LimitExceededCode     = 509
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

// PoolMode is whether PingPong or multiplex
type PoolMode int

const (
	PingPong PoolMode = iota
	Multiplex
	TCP
)

// XFrame represents the minimal programmable object of the protocol.
type XFrame interface {
	// TODO: make multiplexing optional, and maybe we can support PING-PONG protocol in this framework.
	Multiplexing

	HeartbeatPredicate

	// GetTimeout returns a number means Milliseconds for protocol timeout
	// If no timeout defines, returns zero means default timeout
	// If returns a negative number, means never timeout
	GetTimeout() int32

	GetStreamType() StreamType

	GetHeader() HeaderMap

	GetData() IoBuffer

	SetData(data IoBuffer)
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
	Protocol

	Heartbeater

	Hijacker

	PoolMode() PoolMode // configure this to use which connpool

	EnableWorkerPool() bool // same meaning as EnableWorkerPool in types.StreamConnection

	// generate a request id for stream to combine stream request && response
	// use connection param as base
	GenerateRequestID(*uint64) uint64
}

// HeartbeatBuilder provides the ability to construct proper heartbeat command for xprotocol sub-protocols
type Heartbeater interface {
	// Trigger builds an active heartbeat command
	Trigger(context context.Context, requestId uint64) XFrame

	// Reply builds heartbeat command corresponding to the given requestID
	Reply(context context.Context, request XFrame) XRespFrame
}

// Hijacker provides the ability to construct proper response command for xprotocol sub-protocols
type Hijacker interface {
	// BuildResponse build response with given status code
	Hijack(context context.Context, request XFrame, statusCode uint32) XRespFrame

	// Mapping the http status code, which used by proxy framework into protocol-specific status
	Mapping(httpStatusCode uint32) uint32
}

type XProtocolCodec interface {
	ProtocolName() ProtocolName

	XProtocol() XProtocol

	ProtocolMatch() ProtocolMatch

	HTTPMapping() HTTPMapping
}

// XProtocolFactory protocol factory
// Supports automatic creation of a protocol instance for each connection.
// The ability to determine that a protocol is stateful：
// 1. The XProtocolCodec instance implements the XProtocolFactory interface or
// 2. The XProtocol instance implements the XProtocolFactory interface
// The RegisterProtocolFactory function should be called by the developer.
type XProtocolFactory interface {
	NewXProtocol() XProtocol
}
