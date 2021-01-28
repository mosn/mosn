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

import (
	"mosn.io/api/types"
)

// Header key types
const (
	HeaderGlobalTimeout = types.HeaderGlobalTimeout
	HeaderTryTimeout    = types.HeaderTryTimeout
	HeaderOriginalPath  = types.HeaderOriginalPath
)

// Error messages
const (
	ChannelFullException = types.ChannelFullException
	CodecException       = types.CodecException
	SerializeException   = types.SerializeException
	DeserializeException = types.DeserializeException

	NoStatusCodeForHijackException = types.NoStatusCodeForHijackException
)

// Errors
var (
	ErrChanFull             = types.ErrChanFull
	ErrCodecException       = types.ErrCodecException
	ErrSerializeException   = types.ErrSerializeException
	ErrDeserializeException = types.ErrDeserializeException

	ErrNoStatusCodeForHijack = types.ErrNoStatusCodeForHijack
)

// Error codes, used by top level logic code(like proxy logic).
const (
	CodecExceptionCode    = types.CodecExceptionCode
	UnknownCode           = types.UnknownCode
	DeserialExceptionCode = types.DeserialExceptionCode
	SuccessCode           = types.SuccessCode
	PermissionDeniedCode  = types.PermissionDeniedCode
	RouterUnavailableCode = types.RouterUnavailableCode
	InternalErrorCode     = types.InternalErrorCode
	NoHealthUpstreamCode  = types.NoHealthUpstreamCode
	UpstreamOverFlowCode  = types.UpstreamOverFlowCode
	TimeoutExceptionCode  = types.TimeoutExceptionCode
	LimitExceededCode     = types.LimitExceededCode
)

var reason2code = map[StreamResetReason]int{
	StreamConnectionSuccessed: SuccessCode,
	UpstreamGlobalTimeout:     TimeoutExceptionCode,
	UpstreamPerTryTimeout:     TimeoutExceptionCode,
	StreamOverflow:            UpstreamOverFlowCode,
	StreamRemoteReset:         NoHealthUpstreamCode,
	UpstreamReset:             NoHealthUpstreamCode,
	StreamLocalReset:          NoHealthUpstreamCode,
	StreamConnectionFailed:    NoHealthUpstreamCode,
}

// ConvertReasonToCode is convert the reason to a spec code.
func ConvertReasonToCode(reason StreamResetReason) int {
	if code, ok := reason2code[reason]; ok {
		return code
	}

	return InternalErrorCode
}

// ResponseFlags sets
const (
	MosnProcessFailedFlags = types.MosnProcessFailedFlags
)
