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
	"testing"
	mosnctx "sofastack.io/sofa-mosn/pkg/context"
	"sofastack.io/sofa-mosn/pkg/buffer"
	"context"
	"sofastack.io/sofa-mosn/pkg/types"
	"sofastack.io/sofa-mosn/pkg/variable"
	"bufio"
	"bytes"
	"strconv"
)

var (
	postRequestBytes    = []byte("POST /text.json HTTP/1.1\r\nHost: mosn.sofastack.io\r\nScene: http_var_test\r\nContent-Type: text/plain\r\nContent-Length: 13\r\n\r\nHello, World!")
	postRequestBytesLen = len(postRequestBytes)

	getRequestBytes = []byte("GET /info.htm?type=foo&name=bar HTTP/1.1\r\nHost: mosn.sofastack.io\r\nScene: http_var_test\r\nCookie: zone=shanghai\r\nContent-Type: text/plain\r\nContent-Length: 13\r\n\r\nHello, World!")
)

func prepareRequest(t *testing.T, requestBytes []byte) context.Context {
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyListenerPort, 80)
	ctx = buffer.NewBufferPoolContext(ctx)
	ctx = variable.NewVariableContext(ctx)

	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	br := bufio.NewReader(bytes.NewReader(requestBytes))
	err := request.Read(br)
	if err != nil {
		t.Error("parse request failed:", err)
		return nil
	}
	return ctx
}

func Test_get_request_length_and_method(t *testing.T) {
	ctx := prepareRequest(t, postRequestBytes)

	requestLen, err := variable.GetVariableValue(ctx, VarRequestLength)
	if err != nil {
		t.Error("get variable failed:", err)
	}

	expected := strconv.Itoa(postRequestBytesLen)
	if requestLen != expected {
		t.Error("request length assert failed, expected:", expected, ", actual is: ", requestLen)
	}

	requestMethod, err := variable.GetVariableValue(ctx, VarRequestMethod)
	if err != nil {
		t.Error("get variable failed:", err)
	}
	if requestMethod != "POST" {
		t.Error("request method assert failed, expected: POST, actual is: ", requestMethod)
	}
}

func Test_get_header(t *testing.T) {
	ctx := prepareRequest(t, postRequestBytes)

	actual, err := variable.GetVariableValue(ctx, "http_header_scene")
	if err != nil {
		t.Error("get variable failed:", err)
	}

	if actual != "http_var_test" {
		t.Error("request header assert failed, expected: http_var_test, actual is: ", actual)
	}
}

func Test_get_arg(t *testing.T) {
	ctx := prepareRequest(t, getRequestBytes)

	actual, err := variable.GetVariableValue(ctx, "http_arg_type")
	if err != nil {
		t.Error("get variable failed:", err)
	}

	if actual != "foo" {
		t.Error("request arg assert failed, expected: foo, actual is: ", actual)
	}
}

func Test_get_cookie(t *testing.T) {
	ctx := prepareRequest(t, getRequestBytes)

	actual, err := variable.GetVariableValue(ctx, "http_cookie_zone")
	if err != nil {
		t.Error("get variable failed:", err)
	}

	if actual != "shanghai" {
		t.Error("request cookie assert failed, expected: shanghai, actual is: ", actual)
	}
}

func prepareBenchmarkRequest(b *testing.B, requestBytes []byte) context.Context {
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyListenerPort, 80)
	ctx = buffer.NewBufferPoolContext(ctx)
	ctx = variable.NewVariableContext(ctx)

	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	br := bufio.NewReader(bytes.NewReader(requestBytes))
	err := request.Read(br)
	if err != nil {
		b.Error("parse request failed:", err)
		return nil
	}
	return ctx
}

func Benchmark_get_request_length(b *testing.B) {
	ctx := prepareBenchmarkRequest(b, getRequestBytes)

	for i := 0; i < b.N; i++ {
		_, err := variable.GetVariableValue(ctx, "http_request_length")
		if err != nil {
			b.Error("get variable failed:", err)
		}
	}
}

func Benchmark_get_http_header_without_add(b *testing.B) {
	ctx := prepareBenchmarkRequest(b, getRequestBytes)

	for i := 0; i < b.N; i++ {
		_, err := variable.GetVariableValue(ctx, "http_header_scene")
		if err != nil {
			b.Error("get variable failed:", err)
		}
	}
}

func Benchmark_get_http_header_with_add(b *testing.B) {
	variable.AddVariable("http_header_scene")
	ctx := prepareBenchmarkRequest(b, getRequestBytes)

	for i := 0; i < b.N; i++ {
		_, err := variable.GetVariableValue(ctx, "http_header_scene")
		if err != nil {
			b.Error("get variable failed:", err)
		}
	}
}
