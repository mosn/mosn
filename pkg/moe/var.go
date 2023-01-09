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

package moe

import (
	"context"
	"strconv"

	"mosn.io/envoy-go-extension/pkg/api"
	"mosn.io/pkg/variable"

	_ "mosn.io/mosn/pkg/stream/http"
	"mosn.io/mosn/pkg/types"
)

func init() {
	_ = variable.Override(variable.NewStringVariable(types.VarHttpRequestScheme, nil, requestSchemeGetter, nil, 0))
	_ = variable.Override(variable.NewStringVariable(types.VarHttpRequestMethod, nil, requestMethodGetter, nil, 0))
	_ = variable.Override(variable.NewStringVariable(types.VarHttpRequestLength, nil, requestLengthGetter, nil, 0))
	_ = variable.Override(variable.NewStringVariable(types.VarHttpRequestUri, nil, requestUriGetter, nil, 0))
	_ = variable.Override(variable.NewStringVariable(types.VarHttpRequestPath, nil, requestPathGetter, nil, 0))
	_ = variable.Override(variable.NewStringVariable(types.VarHttpRequestPathOriginal, nil, requestPathOriginalGetter, nil, 0))
	_ = variable.Override(variable.NewStringVariable(types.VarHttpRequestArg, nil, requestArgGetter, nil, 0))
}

func requestSchemeGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	// TODO: implement me
	return variable.ValueNotFound, nil
}

func requestMethodGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := streamBufferByContext(ctx)
	if h, ok := buffers.stream.reqHeader.(api.RequestHeaderMap); ok {
		return h.Method(), nil
	}
	return variable.ValueNotFound, nil
}

func requestLengthGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := streamBufferByContext(ctx)

	res := 0
	if h, ok := buffers.stream.reqHeader.(api.RequestHeaderMap); ok {
		res += int(h.ByteSize())
	}
	if b, ok := buffers.stream.reqBody.(api.BufferInstance); ok {
		res += b.Len()
	}

	return strconv.Itoa(res), nil
}

func requestPathGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	buffers := streamBufferByContext(ctx)
	if h, ok := buffers.stream.reqHeader.(api.RequestHeaderMap); ok {
		return h.Path(), nil
	}
	return variable.ValueNotFound, nil
}

func requestPathOriginalGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	// TODO: implement me
	return variable.ValueNotFound, nil
}

func requestUriGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	// TODO: implement me
	return variable.ValueNotFound, nil
}

func requestArgGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	// TODO: implement me
	return variable.ValueNotFound, nil
}
