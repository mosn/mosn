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

package rpc

import (
	"github.com/alipay/sofa-mosn/pkg/types"

	"errors"
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

// RpcCmd act as basic model for different protocols
type RpcCmd interface {
	types.HeaderMap

	ProtocolCode() byte

	RequestID() uint64

	SetRequestID(requestID uint64)

	Header() map[string]string

	Data() types.IoBuffer

	SetHeader(header map[string]string)

	SetData(data types.IoBuffer)
}

// ResponseStatus describe that the model has the [response status] information
type RespStatus interface {
	RespStatus() uint32
}
