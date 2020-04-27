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

package types

import "errors"

// Header key types
const (
	HeaderStatus                   = "x-mosn-status"
	HeaderMethod                   = "x-mosn-method"
	HeaderHost                     = "x-mosn-host"
	HeaderPath                     = "x-mosn-path"
	HeaderQueryString              = "x-mosn-querystring"
	HeaderStreamID                 = "x-mosn-streamid"
	HeaderGlobalTimeout            = "x-mosn-global-timeout"
	HeaderTryTimeout               = "x-mosn-try-timeout"
	HeaderException                = "x-mosn-exception"
	HeaderStremEnd                 = "x-mosn-endstream"
	HeaderRPCService               = "x-mosn-rpc-service"
	HeaderRPCMethod                = "x-mosn-rpc-method"
	HeaderXprotocolSubProtocol     = "x-mosn-xprotocol-sub-protocol"
	HeaderXprotocolStreamId        = "x-mosn-xprotocol-stream-id"
	HeaderXprotocolRespStatus      = "x-mosn-xprotocol-resp-status"
	HeaderXprotocolRespIsException = "x-mosn-xprotocol-resp-is-exception"
	HeaderXprotocolHeartbeat       = "x-protocol-heartbeat"
)

// Error messages
const (
	ChannelFullException = "Channel is full"
	CodecException       = "codec exception occurs"
	SerializeException   = "serialize exception occurs"
	DeserializeException = "deserialize exception occurs"

	NoStatusCodeForHijackException = "no status code found for hijack reply"
)

// Errors
var (
	ErrChanFull             = errors.New(ChannelFullException)
	ErrCodecException       = errors.New(CodecException)
	ErrSerializeException   = errors.New(SerializeException)
	ErrDeserializeException = errors.New(DeserializeException)

	ErrNoStatusCodeForHijack = errors.New(NoStatusCodeForHijackException)
)

// Error codes, used by top level logic code(like proxy logic).
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

// ConvertReasonToCode is convert the reason to a spec code.
func ConvertReasonToCode(reason StreamResetReason) int {
	switch reason {
	case StreamConnectionSuccessed:
		return SuccessCode

	case UpstreamGlobalTimeout, UpstreamPerTryTimeout:
		return TimeoutExceptionCode

	case StreamOverflow:
		return UpstreamOverFlowCode

	case StreamRemoteReset, UpstreamReset, StreamLocalReset, StreamConnectionFailed:
		return NoHealthUpstreamCode

	default:
		return InternalErrorCode
	}
}
