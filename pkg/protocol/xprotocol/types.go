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

// XStream
type StreamType int

const (
	Request StreamType = iota
	RequestOneWay
	Response
)

var (
	AlreadyRegistered = "protocol code already registered."
	UnknownType       = "unknown model type."
	UnrecognizedCode  = "unrecognized protocol code."
	NoProtocolCode    = "no protocol code found."

	ErrDupRegistered    = errors.New(AlreadyRegistered)
	ErrUnknownType      = errors.New(UnknownType)
	ErrUnrecognizedCode = errors.New(UnrecognizedCode)
	ErrNoProtocolCode   = errors.New(NoProtocolCode)
)

type XFrame interface {
	Multiplexing

	HeartbeatPredicate

	GetStreamType() StreamType

	GetHeader() types.HeaderMap

	GetData() types.IoBuffer
}

type XRespFrame interface {
	XFrame

	GetStatusCode() uint32
}

type Multiplexing interface {
	GetRequestId() uint64

	SetRequestId(id uint64)
}

type HeartbeatPredicate interface {
	IsHeartbeatFrame() bool
}

type HeaderMutator interface {
	MutateHeader(bytes []byte) []byte
}

type ServiceAware interface {
	GetServiceName() string

	GetMethodName() string
}

type GoAwayPredicate interface {
	IsGoAwayFrame() bool
}

// XProtocol
type XProtocol interface {
	types.Protocol

	Heartbeater

	Hijacker
}

// HeartbeatBuilder provides interface to construct proper heartbeat command for xprotocol sub-protocols
type Heartbeater interface {
	// Trigger builds an active heartbeat command
	Trigger(requestId uint64) XFrame

	// Reply builds heartbeat command corresponding to the given requestID
	Reply(requestId uint64) XRespFrame
}

// HeartbeatBuilder provides interface to construct proper response command for xprotocol sub-protocols
type Hijacker interface {
	// BuildResponse build response with given status code
	Hijack(statusCode uint32) XRespFrame

	Mapping(httpStatusCode uint32) uint32
}
