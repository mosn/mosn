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

package httpconv

import (
	"context"
	"errors"
	"strings"

	"github.com/valyala/fasthttp"
	"mosn.io/api"
	apitran "mosn.io/api/extensions/transcoder"
	"mosn.io/mosn/pkg/filter/stream/transcoder"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

var errProtocolNotRequired = errors.New("protocol is not the required")

func init() {
	transcoder.MustRegister("httpTohttp2", NewHttpToHttp2)
}

type httpTohttp2 struct {
}

func NewHttpToHttp2(config map[string]interface{}) apitran.Transcoder {
	return &httpTohttp2{}
}

// Accept check the request will be transcoded or not
// http request will be translated
func (t *httpTohttp2) Accept(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) bool {
	_, ok := headers.(http.RequestHeader)
	return ok
}

// TranscodingRequest makes http request to http2 request
func (t *httpTohttp2) TranscodingRequest(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) (api.HeaderMap, api.IoBuffer, api.HeaderMap, error) {
	httpHeader, ok := headers.(http.RequestHeader)
	if !ok {
		return headers, buf, trailers, errProtocolNotRequired
	}
	// convert http header to common header, the http2 stream will encode common header to http2 header
	cheader := make(map[string]string, httpHeader.Len())
	httpHeader.VisitAll(func(key, value []byte) {
		cheader[strings.ToLower(string(key))] = string(value)
	})

	// set upstream protocol
	_ = variable.Set(ctx, types.VariableUpstreamProtocol, protocol.HTTP2)

	return protocol.CommonHeader(cheader), buf, trailers, nil
}

// TranscodingResponse make http2 response to http repsonse
func (t *httpTohttp2) TranscodingResponse(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) (api.HeaderMap, api.IoBuffer, api.HeaderMap, error) {
	if _, ok := headers.(*http2.RspHeader); !ok {
		// if the response is not bolt response, it maybe come from hijack or send directly response.
		// so we just returns the original data
		return headers, buf, trailers, nil
	}
	cheader := http2.DecodeHeader(headers)
	headerImpl := http.ResponseHeader{&fasthttp.ResponseHeader{}}
	cheader.Range(func(key, value string) bool {
		headerImpl.Set(key, value)
		return true
	})
	return headerImpl, buf, trailers, nil
}
