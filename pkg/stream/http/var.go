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
	"sofastack.io/sofa-mosn/pkg/variable"
	"context"
	"strconv"
	"strings"
)

var (
	httpVariables = []variable.Variable{
		variable.NewSimpleBasicVariable("http_request_length", getRequestBytes, variable.MOSN_VAR_FLAG_INDEXED),
	}
)

func init() {
	// register built-in variables
	for idx := range httpVariables {
		variable.RegisterVariable(httpVariables[idx])
	}

	// register prefix getter
	variable.RegisterPrefixVariable("http_header_", getHttpHeader)
}

func getRequestBytes(ctx context.Context, data interface{}) string {
	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest
	return strconv.Itoa(len(request.Header.Header()) + len(request.Body()))
}

func getHttpHeader(ctx context.Context, data interface{}) string {
	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	headerName := data.(string)
	return string(request.Header.Peek(strings.TrimPrefix(headerName, "http_header_")))
}
