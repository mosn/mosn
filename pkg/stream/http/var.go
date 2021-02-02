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

package http

import (
	"context"
	"strconv"

	"mosn.io/api"
	"mosn.io/pkg/variable"

	"mosn.io/mosn/pkg/protocol"
)

const (
	VarRequestMethod = "http_request_method"
	VarRequestLength = "http_request_length"
)

var (
	headerIndex = len(variable.VarPrefixHttpHeader)
	cookieIndex = len(variable.VarPrefixHttpCookie)
	argIndex    = len(variable.VarPrefixHttpArg)

	builtinVariables = []variable.Variable{
		variable.NewBasicVariable(variable.VarHttpRequestScheme, nil, requestSchemeGetter, nil, 0),
		variable.NewBasicVariable(variable.VarHttpRequestMethod, nil, requestMethodGetter, nil, 0),
		variable.NewBasicVariable(variable.VarHttpRequestLength, nil, requestLengthGetter, nil, 0),
		variable.NewBasicVariable(variable.VarHttpRequestUri, nil, requestUriGetter, nil, 0),
		variable.NewBasicVariable(variable.VarHttpRequestPath, nil, requestPathGetter, nil, 0),
		variable.NewBasicVariable(variable.VarHttpRequestArg, nil, requestArgGetter, nil, 0),
	}

	prefixVariables = []variable.Variable{
		variable.NewBasicVariable(variable.VarPrefixHttpHeader, nil, httpHeaderGetter, nil, 0),
		variable.NewBasicVariable(variable.VarPrefixHttpArg, nil, httpArgGetter, nil, 0),
		variable.NewBasicVariable(variable.VarPrefixHttpCookie, nil, httpCookieGetter, nil, 0),
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

	// register protocol resource
	variable.RegisterProtocolResource(protocol.HTTP1, api.SCHEME, variable.VarProtocolRequestScheme)
	variable.RegisterProtocolResource(protocol.HTTP1, api.PATH, variable.VarProtocolRequestPath)
	variable.RegisterProtocolResource(protocol.HTTP1, api.URI, variable.VarProtocolRequestUri)
	variable.RegisterProtocolResource(protocol.HTTP1, api.ARG, variable.VarProtocolRequestArg)
	variable.RegisterProtocolResource(protocol.HTTP1, api.COOKIE, variable.VarProtocolCookie)
	variable.RegisterProtocolResource(protocol.HTTP1, api.HEADER, variable.VarProtocolRequestHeader)
}

func requestSchemeGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := httpBuffersByContext(ctx)
	return string(buffers.serverRequest.URI().Scheme()), nil
}

func requestMethodGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	return string(request.Header.Method()), nil
}

func requestLengthGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	length := len(request.Header.Header()) + len(request.Body())
	if length == 0 {
		return variable.ValueNotFound, nil
	}

	return strconv.Itoa(length), nil
}

func requestPathGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	return string(request.URI().Path()), nil
}

func requestUriGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	return string(request.Header.RequestURI()), nil
}

func requestArgGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest
	return request.URI().QueryArgs().String(), nil
}

func httpHeaderGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	headerName := data.(string)
	headerValue := request.Header.Peek(headerName[headerIndex:])
	// nil means no kv exists, "" means kv exists, but value is ""
	if headerValue == nil {
		return variable.ValueNotFound, nil
	}

	return string(headerValue), nil
}

func httpArgGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	argName := data.(string)
	// TODO: support post args
	argValue := request.URI().QueryArgs().Peek(argName[argIndex:])
	// nil means no kv exists, "" means kv exists, but value is ""
	if argValue == nil {
		return variable.ValueNotFound, nil
	}

	return string(argValue), nil
}

func httpCookieGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	cookieName := data.(string)
	cookieValue := request.Header.Cookie(cookieName[cookieIndex:])
	// nil means no kv exists, "" means kv exists, but value is ""
	if cookieValue == nil {
		return variable.ValueNotFound, nil
	}

	return string(cookieValue), nil
}
