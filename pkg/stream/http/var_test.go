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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

var (
	postRequestBytes    = []byte("POST /text.json HTTP/1.1\r\nHost: mosn.sofastack.io\r\nScene: http_var_test\r\nContent-Type: text/plain\r\nContent-Length: 13\r\n\r\nHello, World!")
	postRequestBytesLen = len(postRequestBytes)

	getRequestBytes = []byte("GET /info.htm?type=foo&name=bar HTTP/1.1\r\nHost: mosn.sofastack.io\r\nScene: http_var_test\r\nCookie: zone=shanghai\r\nContent-Type: text/plain\r\nContent-Length: 13\r\n\r\nHello, World!")
)

func prepareRequest(t *testing.T, requestBytes []byte) context.Context {
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableListenerPort, 80)
	ctx = buffer.NewBufferPoolContext(ctx)

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

func Test_get_scheme(t *testing.T) {
	ctx := prepareRequest(t, getRequestBytes)

	actual, err := variable.GetString(ctx, fmt.Sprintf("%s_%s", protocol.HTTP1, types.VarProtocolRequestScheme))
	if err != nil {
		t.Error("get variable failed:", err)
	}

	want := "http"
	if actual != want {
		t.Errorf("request scheme assert failed, expected: %s, actual is: %s", want, actual)
	}
}

func Test_get_request_length_and_method(t *testing.T) {
	ctx := prepareRequest(t, postRequestBytes)

	requestLen, err := variable.GetString(ctx, types.VarHttpRequestLength)
	if err != nil {
		t.Error("get variable failed:", err)
	}

	expected := strconv.Itoa(postRequestBytesLen)
	if requestLen != expected {
		t.Error("request length assert failed, expected:", expected, ", actual is: ", requestLen)
	}

	requestMethod, err := variable.GetString(ctx, types.VarHttpRequestMethod)
	if err != nil {
		t.Error("get variable failed:", err)
	}
	if requestMethod != "POST" {
		t.Error("request method assert failed, expected: POST, actual is: ", requestMethod)
	}
}

func Test_get_header(t *testing.T) {
	ctx := prepareRequest(t, postRequestBytes)

	actual, err := variable.GetString(ctx,
		fmt.Sprintf("%s_%s%s", protocol.HTTP1, types.VarProtocolRequestHeader, "scene"))
	if err != nil {
		t.Error("get variable failed:", err)
	}

	if actual != "http_var_test" {
		t.Error("request header assert failed, expected: http_var_test, actual is: ", actual)
	}

	_, err = variable.GetString(ctx,
		fmt.Sprintf("%s_%s%s", protocol.HTTP1, types.VarProtocolRequestHeader, "err_test"))
	if err == nil {
		t.Error("request header assert failed, should get an err, actually get a nil")
	}

	if err.Error() != variable.ErrValueNotFound.Error() {
		t.Errorf("request header assert failed, the err message should be %s", variable.ErrValueNotFound.Error())
	}
}

func Test_get_arg(t *testing.T) {
	ctx := prepareRequest(t, getRequestBytes)

	actual, err := variable.GetString(ctx,
		fmt.Sprintf("%s_%s%s", protocol.HTTP1, types.VarProtocolRequestArgPrefix, "type"))
	if err != nil {
		t.Error("get variable failed:", err)
	}

	if actual != "foo" {
		t.Error("request arg assert failed, expected: foo, actual is: ", actual)
	}

	_, err = variable.GetString(ctx,
		fmt.Sprintf("%s_%s%s", protocol.HTTP1, types.VarProtocolRequestArgPrefix, "err_test"))
	if err == nil {
		t.Error("request arg assert failed, should get an err, actually get a nil")
	}

	if err.Error() != variable.ErrValueNotFound.Error() {
		t.Errorf("request arg assert failed, the err message should be %s", variable.ErrValueNotFound.Error())
	}
}

func Test_get_cookie(t *testing.T) {
	ctx := prepareRequest(t, getRequestBytes)

	actual, err := variable.GetString(ctx,
		fmt.Sprintf("%s%s", types.VarPrefixHttpCookie, "zone"))
	if err != nil {
		t.Error("get variable failed:", err)
	}

	if actual != "shanghai" {
		t.Error("request cookie assert failed, expected: shanghai, actual is: ", actual)
	}

	_, err = variable.GetString(
		ctx,
		fmt.Sprintf("%s%s", types.VarPrefixHttpCookie, "err_test"),
	)

	if err == nil {
		t.Error("request cookie assert failed, should get an err, actually get a nil")
	}
	if err.Error() != variable.ErrValueNotFound.Error() {
		t.Errorf("request cookie assert failed, the err message should be %s", variable.ErrValueNotFound.Error())
	}
}

func Test_get_path(t *testing.T) {
	ctx := prepareRequest(t, getRequestBytes)

	actual, err := variable.GetString(ctx,
		fmt.Sprintf("%s_%s", protocol.HTTP1, types.VarProtocolRequestPath))
	if err != nil {
		t.Error("get variable failed:", err)
	}

	want := "/info.htm"
	if actual != want {
		t.Errorf("request path assert failed, expected: %s, actual is: %s", want, actual)
	}
}

func Test_get_uri(t *testing.T) {
	ctx := prepareRequest(t, getRequestBytes)

	actual, err := variable.GetString(ctx,
		fmt.Sprintf("%s_%s", protocol.HTTP1, types.VarProtocolRequestUri))
	if err != nil {
		t.Error("get variable failed:", err)
	}

	want := "/info.htm?type=foo&name=bar"
	if actual != want {
		t.Errorf("request uri assert failed, expected: %s, actual is: %s", want, actual)
	}
}

func Test_get_allarg(t *testing.T) {
	ctx := prepareRequest(t, getRequestBytes)

	actual, err := variable.GetString(ctx,
		fmt.Sprintf("%s_%s", protocol.HTTP1, types.VarProtocolRequestArg))
	if err != nil {
		t.Error("get variable failed:", err)
	}
	want := "type=foo&name=bar"
	if actual != want {
		t.Errorf("request arg assert failed, expected: %s, actual is: %s", want, actual)
	}
}

func Test_get_protocolResource(t *testing.T) {
	ctx := prepareRequest(t, getRequestBytes)

	_ = variable.Set(ctx, types.VariableDownStreamProtocol, protocol.HTTP1)
	actual, err := variable.GetProtocolResource(ctx, api.PATH)
	if err != nil {
		t.Error("get variable failed:", err)
	}
	want := "/info.htm"
	if actual != want {
		t.Errorf("request arg assert failed, expected: %s, actual is: %s", want, actual)
	}

	actual, err = variable.GetProtocolResource(ctx, api.URI)
	if err != nil {
		t.Error("get variable failed:", err)
	}
	want = "/info.htm?type=foo&name=bar"
	if actual != want {
		t.Errorf("request arg assert failed, expected: %s, actual is: %s", want, actual)
	}

	actual, err = variable.GetProtocolResource(ctx, api.ARG)
	if err != nil {
		t.Error("get variable failed:", err)
	}
	want = "type=foo&name=bar"
	if actual != want {
		t.Errorf("request arg assert failed, expected: %s, actual is: %s", want, actual)
	}
}

func Test_getPrefixProtocolVarHeaderAndCookie(t *testing.T) {
	ctx := prepareRequest(t, getRequestBytes)
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, protocol.HTTP1)

	cookieName := "zone"
	expect := "shanghai"
	actual, err := variable.GetProtocolResource(ctx, api.COOKIE, cookieName)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !assert.Equalf(t, expect, actual, "cookie value expect to be %s, but get %s") {
		t.FailNow()
	}

	headerName := "Content-Type"
	expect = "text/plain"
	actual, err = variable.GetProtocolResource(ctx, api.HEADER, headerName)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !assert.Equalf(t, expect, actual, "header value expect to be %s, but get %s") {
		t.FailNow()
	}
}

func prepareBenchmarkRequest(b *testing.B, requestBytes []byte) context.Context {
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableListenerPort, 80)
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, protocol.HTTP1)

	ctx = buffer.NewBufferPoolContext(ctx)

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
		_, err := variable.GetString(ctx, "Http1_request_length")
		if err != nil {
			b.Error("get variable failed:", err)
		}
	}
}
