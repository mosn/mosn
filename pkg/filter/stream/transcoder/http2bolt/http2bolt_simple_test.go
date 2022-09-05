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

package http2bolt

import (
	"context"
	"reflect"
	"testing"

	"github.com/valyala/fasthttp"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/protocol/http"
)

func Test_http2bolt_Accept(t1 *testing.T) {
	type args struct {
		ctx      context.Context
		headers  types.HeaderMap
		buf      types.IoBuffer
		trailers types.HeaderMap
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "http header",
			args: args{
				ctx:      context.Background(),
				headers:  http.RequestHeader{RequestHeader: &fasthttp.RequestHeader{}},
				buf:      buffer.NewIoBufferString("Test_http2bolt_Accept"),
				trailers: nil,
			},
			want: true,
		},
		{
			name: "non http header",
			args: args{
				ctx:      context.Background(),
				headers:  &bolt.Request{},
				buf:      buffer.NewIoBufferString("Test_http2bolt_Accept"),
				trailers: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &http2bolt{}
			if got := t.Accept(tt.args.ctx, tt.args.headers, tt.args.buf, tt.args.trailers); got != tt.want {
				t1.Errorf("Accept() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_http2bolt_TranscodingRequest(t1 *testing.T) {
	bufData := buffer.NewIoBufferString("Test_http2bolt_TranscodingRequest")

	type args struct {
		ctx      context.Context
		headers  types.HeaderMap
		buf      types.IoBuffer
		trailers types.HeaderMap
	}
	tests := []struct {
		name     string
		args     args
		wantArgs args
		wantErr  bool
	}{
		{
			name: "normal",
			args: args{
				ctx:      context.Background(),
				headers:  buildHttpRequestHeaders(map[string]string{"Service": "test", "Scene": "ut"}),
				buf:      bufData,
				trailers: nil,
			},
			wantArgs: args{
				headers:  bolt.NewRpcRequest(0, protocol.CommonHeader(map[string]string{"Service": "test", "Scene": "ut"}), bufData),
				buf:      bufData,
				trailers: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &http2bolt{}
			got, got1, got2, err := t.TranscodingRequest(tt.args.ctx, tt.args.headers, tt.args.buf, tt.args.trailers)
			if (err != nil) != tt.wantErr {
				t1.Errorf("TranscodingRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !checkHeadersEqual(got, tt.wantArgs.headers) {
				t1.Errorf("TranscodingRequest() headers got = %v, want %v", got, tt.wantArgs.headers)
			}
			if !reflect.DeepEqual(got1, tt.wantArgs.buf) {
				t1.Errorf("TranscodingRequest() buf got = %v, want %v", got1, tt.wantArgs.buf)
			}
			if !reflect.DeepEqual(got2, tt.wantArgs.trailers) {
				t1.Errorf("TranscodingRequest() trailers got = %v, want %v", got2, tt.wantArgs.trailers)
			}
		})
	}
}

func Test_http2bolt_TranscodingResponse(t1 *testing.T) {
	bufData := buffer.NewIoBufferString("Test_http2bolt_TranscodingResponse")

	type args struct {
		ctx      context.Context
		headers  types.HeaderMap
		buf      types.IoBuffer
		trailers types.HeaderMap
	}
	tests := []struct {
		name     string
		args     args
		wantArgs args
		wantErr  bool
	}{
		{
			name: "normal_success",
			args: args{
				ctx:      context.Background(),
				headers:  bolt.NewRpcResponse(0, bolt.ResponseStatusSuccess, protocol.CommonHeader(map[string]string{"service": "test", "scene": "ut"}), bufData),
				buf:      bufData,
				trailers: nil,
			},
			wantArgs: args{
				headers:  buildHttpResponseHeaders(http.OK, map[string]string{"Service": "test", "Scene": "ut"}),
				buf:      bufData,
				trailers: nil,
			},
			wantErr: false,
		},
		{
			name: "normal_internal_server_error",
			args: args{
				ctx:      context.Background(),
				headers:  bolt.NewRpcResponse(0, bolt.ResponseStatusServerException, protocol.CommonHeader(map[string]string{"service": "test", "scene": "ut"}), buffer.NewIoBufferString("Test_http2bolt_TranscodingResponse")),
				buf:      buffer.NewIoBufferString("Test_http2bolt_TranscodingResponse"),
				trailers: nil,
			},
			wantArgs: args{
				headers:  buildHttpResponseHeaders(http.InternalServerError, map[string]string{"Service": "test", "Scene": "ut"}),
				buf:      buffer.NewIoBufferString("Test_http2bolt_TranscodingResponse"),
				trailers: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &http2bolt{}
			got, got1, got2, err := t.TranscodingResponse(tt.args.ctx, tt.args.headers, tt.args.buf, tt.args.trailers)
			if (err != nil) != tt.wantErr {
				t1.Errorf("TranscodingResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !checkHeadersEqual(got, tt.wantArgs.headers) {
				t1.Errorf("TranscodingResponse() headers got = %v, want %v", got, tt.wantArgs.headers)
			}
			if !reflect.DeepEqual(got1, tt.wantArgs.buf) {
				t1.Errorf("TranscodingResponse() buf got = %v, want %v", got1, tt.wantArgs.buf)
			}
			if !reflect.DeepEqual(got2, tt.wantArgs.trailers) {
				t1.Errorf("TranscodingResponse() trailers got = %v, want %v", got2, tt.wantArgs.trailers)
			}
		})
	}
}

func buildHttpRequestHeaders(args map[string]string) http.RequestHeader {
	header := &fasthttp.RequestHeader{}

	for key, value := range args {
		header.Set(key, value)
	}

	return http.RequestHeader{RequestHeader: header}
}

func buildHttpResponseHeaders(status int, args map[string]string) http.ResponseHeader {
	header := &fasthttp.ResponseHeader{}

	for key, value := range args {
		header.Set(key, value)
	}

	header.SetStatusCode(status)

	return http.ResponseHeader{ResponseHeader: header}
}

func checkHeadersEqual(left, right types.HeaderMap) bool {
	leftFlat := make(map[string]string)
	rightFlat := make(map[string]string)

	left.Range(func(key, value string) bool {
		leftFlat[key] = value
		return true
	})

	right.Range(func(key, value string) bool {
		rightFlat[key] = value
		return true
	})

	headerEqual := reflect.DeepEqual(leftFlat, rightFlat)
	statusEqual := true

	switch left.(type) {
	case http.ResponseHeader:
		leftHttpResp := left.(http.ResponseHeader)
		rightHttpResp := right.(http.ResponseHeader)

		statusEqual = leftHttpResp.StatusCode() == rightHttpResp.StatusCode()
	}

	return headerEqual && statusEqual
}
