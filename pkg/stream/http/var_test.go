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

func Test_get_request_length(t *testing.T) {
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyListenerPort, 80)
	ctx = buffer.NewBufferPoolContext(ctx)
	ctx = variable.NewVariableContext(ctx)

	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	requestBytes := []byte("POST /text.json HTTP/1.1\r\nHost: mosn.sofastack.io\r\nTest-Method: Test_get_request_length\r\nContent-Type: text/plain\r\nContent-Length: 13\r\n\r\nHello, World!")
	requestBytesLen := len(requestBytes)
	br := bufio.NewReader(bytes.NewReader(requestBytes))
	err := request.Read(br)
	if err != nil {
		t.Error("parse request failed:", err)
	}

	requestLen, err := variable.GetVariableValue(ctx, "http_request_length")
	if err != nil {
		t.Error("get variable failed:", err)
	}

	expected := strconv.Itoa(requestBytesLen)
	if requestLen != expected {
		t.Error("request length assert failed, expected:", expected, ", actual is: ", requestLen)
	}
}

func Test_get_header(t *testing.T) {
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyListenerPort, 80)
	ctx = buffer.NewBufferPoolContext(ctx)
	ctx = variable.NewVariableContext(ctx)

	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	requestBytes := []byte("POST /text.json HTTP/1.1\r\nHost: mosn.sofastack.io\r\nTest-Method: Test_get_header\r\nContent-Type: text/plain\r\nContent-Length: 13\r\n\r\nHello, World!")
	br := bufio.NewReader(bytes.NewReader(requestBytes))
	err := request.Read(br)
	if err != nil {
		t.Error("parse request failed:", err)
	}

	actual, err := variable.GetVariableValue(ctx, "http_header_test-method")
	if err != nil {
		t.Error("get variable failed:", err)
	}

	if actual != "Test_get_header" {
		t.Error("request length assert failed, expected: Test_get_header, actual is: ", actual)
	}
}

func Benchmark_get_request_length(b *testing.B) {
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyListenerPort, 80)
	ctx = buffer.NewBufferPoolContext(ctx)
	ctx = variable.NewVariableContext(ctx)

	buffers := httpBuffersByContext(ctx)
	request := &buffers.serverRequest

	requestBytes := []byte("POST /text.json HTTP/1.1\r\nHost: mosn.sofastack.io\r\nTest-Method: Test_get_request_length\r\nContent-Type: text/plain\r\nContent-Length: 13\r\n\r\nHello, World!")
	br := bufio.NewReader(bytes.NewReader(requestBytes))
	err := request.Read(br)
	if err != nil {
		b.Error("parse request failed:", err)
	}

	for i := 0; i < b.N; i++ {
		_, err := variable.GetVariableValue(ctx, "http_request_length")
		if err != nil {
			b.Error("get variable failed:", err)
		}
	}
}
