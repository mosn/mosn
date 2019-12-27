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

package proxy

import (
	"context"
	"strconv"

	"mosn.io/mosn/pkg/variable"
)

// The identification of a request info's content
const (
	VarStartTime                string = "start_time"
	VarRequestReceivedDuration  string = "request_received_duration"
	VarResponseReceivedDuration string = "response_received_duration"
	VarRequestFinishedDuration  string = "request_finished_duration"
	VarBytesSent                string = "bytes_sent"
	VarBytesReceived            string = "bytes_received"
	VarProtocol                 string = "protocol"
	VarResponseCode             string = "response_code"
	VarDuration                 string = "duration"
	VarResponseFlag             string = "response_flag"
	VarUpstreamLocalAddress     string = "upstream_local_address"
	VarDownstreamLocalAddress   string = "downstream_local_address"
	VarDownstreamRemoteAddress  string = "downstream_remote_address"
	VarUpstreamHost             string = "upstream_host"

	// ReqHeaderPrefix is the prefix of request header's formatter
	reqHeaderPrefix string = "request_header_"
	reqHeaderIndex         = len(reqHeaderPrefix)
	// RespHeaderPrefix is the prefix of response header's formatter
	respHeaderPrefix string = "response_header_"
	respHeaderIndex         = len(respHeaderPrefix)
)

var (
	builtinVariables = []variable.Variable{
		variable.NewBasicVariable(VarStartTime, nil, startTimeGetter, nil, 0),
		variable.NewBasicVariable(VarRequestReceivedDuration, nil, receivedDurationGetter, nil, 0),
		variable.NewBasicVariable(VarResponseReceivedDuration, nil, responseReceivedDurationGetter, nil, 0),
		variable.NewBasicVariable(VarRequestFinishedDuration, nil, requestFinishedDurationGetter, nil, 0),
		variable.NewBasicVariable(VarBytesSent, nil, bytesSentGetter, nil, 0),
		variable.NewBasicVariable(VarBytesReceived, nil, bytesReceivedGetter, nil, 0),
		variable.NewBasicVariable(VarProtocol, nil, protocolGetter, nil, 0),
		variable.NewBasicVariable(VarResponseCode, nil, responseCodeGetter, nil, 0),
		variable.NewBasicVariable(VarDuration, nil, durationGetter, nil, 0),
		variable.NewBasicVariable(VarResponseFlag, nil, responseFlagGetter, nil, 0),
		variable.NewBasicVariable(VarUpstreamLocalAddress, nil, upstreamLocalAddressGetter, nil, 0),
		variable.NewBasicVariable(VarDownstreamLocalAddress, nil, downstreamLocalAddressGetter, nil, 0),
		variable.NewBasicVariable(VarDownstreamRemoteAddress, nil, downstreamRemoteAddressGetter, nil, 0),
		variable.NewBasicVariable(VarUpstreamHost, nil, upstreamHostGetter, nil, 0),
	}

	prefixVariables = []variable.Variable{
		variable.NewBasicVariable(reqHeaderPrefix, nil, requestHeaderMapGetter, nil, 0),
		variable.NewBasicVariable(respHeaderPrefix, nil, responseHeaderMapGetter, nil, 0),
	}
)

func init() {
	// register built-in variables
	for idx := range builtinVariables {
		variable.RegisterVariable(builtinVariables[idx])
	}

	// register prefix variables, like header_xxx/arg_xxx/cookie_xxx
	for idx := range prefixVariables {
		variable.RegisterPrefixVariable(prefixVariables[idx].Name(), prefixVariables[idx])
	}
}

// StartTimeGetter
// get request's arriving time
func startTimeGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return info.StartTime().Format("2006/01/02 15:04:05.000"), nil
}

// ReceivedDurationGetter
// get duration between request arriving and request resend to upstream
func receivedDurationGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return info.RequestReceivedDuration().String(), nil
}

// ResponseReceivedDurationGetter
// get duration between request arriving and response sending
func responseReceivedDurationGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return info.ResponseReceivedDuration().String(), nil
}

// RequestFinishedDurationGetter hets duration between request arriving and request finished
func requestFinishedDurationGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return info.RequestFinishedDuration().String(), nil
}

// BytesSentGetter
// get bytes sent
func bytesSentGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return strconv.FormatUint(info.BytesSent(), 10), nil
}

// BytesReceivedGetter
// get bytes received
func bytesReceivedGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return strconv.FormatUint(info.BytesReceived(), 10), nil
}

// get request's protocol type
func protocolGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return string(info.Protocol()), nil
}

// ResponseCodeGetter
// get request's response code
func responseCodeGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return strconv.FormatUint(uint64(info.ResponseCode()), 10), nil
}

// DurationGetter
// get duration since request's starting time
func durationGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return info.Duration().String(), nil
}

// GetResponseFlagGetter
// get request's response flag
func responseFlagGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return strconv.FormatBool(info.GetResponseFlag(0)), nil
}

// UpstreamLocalAddressGetter
// get upstream's local address
func upstreamLocalAddressGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	return info.UpstreamLocalAddress(), nil
}

// DownstreamLocalAddressGetter
// get downstream's local address
func downstreamLocalAddressGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	if info.DownstreamLocalAddress() != nil {
		return info.DownstreamLocalAddress().String(), nil
	}

	return variable.ValueNotFound, nil
}

// DownstreamRemoteAddressGetter
// get upstream's remote address
func downstreamRemoteAddressGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	if info.DownstreamRemoteAddress() != nil {
		return info.DownstreamRemoteAddress().String(), nil
	}

	return variable.ValueNotFound, nil
}

// upstreamHostGetter
// get upstream's selected host address
func upstreamHostGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	info := proxyBuffers.info

	if info.UpstreamHost() != nil {
		return info.UpstreamHost().Hostname(), nil
	}

	return variable.ValueNotFound, nil
}

func requestHeaderMapGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	headers := proxyBuffers.stream.downstreamReqHeaders

	headerName := data.(string)
	headerValue, ok := headers.Get(headerName[reqHeaderIndex:])
	if !ok {
		return variable.ValueNotFound, nil
	}

	return string(headerValue), nil
}

func responseHeaderMapGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	proxyBuffers := proxyBuffersByContext(ctx)
	headers := proxyBuffers.request.upstreamRespHeaders

	headerName := data.(string)
	headerValue, ok := headers.Get(headerName[respHeaderIndex:])
	if !ok {
		return variable.ValueNotFound, nil
	}

	return string(headerValue), nil
}
