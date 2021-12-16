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
	"strings"

	"github.com/valyala/fasthttp"
	"mosn.io/api"
	apitran "mosn.io/api/extensions/transcoder"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/filter/stream/transcoder"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
)

func init() {
	transcoder.MustRegister("http2Tohttp", NewHttpToHttp)
}

type http2Tohttp struct {
}

func NewHttpToHttp(config map[string]interface{}) apitran.Transcoder {
	return &http2Tohttp{}
}

func (t *http2Tohttp) Accept(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) bool {
	_, ok := headers.(*http2.ReqHeader)
	return ok
}

// TranscodingRequest makes a http2 request to http request
func (t *http2Tohttp) TranscodingRequest(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) (api.HeaderMap, api.IoBuffer, api.HeaderMap, error) {
	httpHeader := http2.DecodeHeader(headers)
	headerImpl := http.RequestHeader{&fasthttp.RequestHeader{}}

	httpHeader.Range(func(key, value string) bool {
		headerImpl.Set(key, value)
		return true
	})

	mosnctx.WithValue(ctx, types.ContextKeyUpStreamProtocol, string(protocol.HTTP1))

	return headerImpl, buf, trailers, nil
}

// TranscodingResponse makes a http resposne to http2 response
func (t *http2Tohttp) TranscodingResponse(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) (api.HeaderMap, api.IoBuffer, api.HeaderMap, error) {
	httpHeader, ok := headers.(http.ResponseHeader)
	if !ok {
		// if the response is not bolt response, it maybe come from hijack or send directly response.
		// so we just returns the original data
		return headers, buf, trailers, nil
	}
	cheader := make(map[string]string, httpHeader.Len())
	// copy headers
	httpHeader.VisitAll(func(key, value []byte) {
		cheader[strings.ToLower(string(key))] = string(value)
	})

	return protocol.CommonHeader(cheader), buf, trailers, nil
}
