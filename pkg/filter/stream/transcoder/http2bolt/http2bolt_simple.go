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

	"github.com/valyala/fasthttp"
	apit "mosn.io/api/extensions/transcoder"
	"mosn.io/mosn/pkg/filter/stream/transcoder"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/protocol/http"
	"mosn.io/pkg/variable"
)

func init() {
	transcoder.MustRegister("http2bolt_simple", NewTranscoder)
}

type http2bolt struct {
}

func NewTranscoder(config map[string]interface{}) apit.Transcoder {
	return &http2bolt{}
}

func (t *http2bolt) Accept(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) bool {
	_, ok := headers.(http.RequestHeader)
	return ok
}

func (t *http2bolt) TranscodingRequest(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) (types.HeaderMap, types.IoBuffer, types.HeaderMap, error) {
	// 1.set upstream protocol
	_ = variable.Set(ctx, types.VariableUpstreamProtocol, bolt.ProtocolName)
	// 2. assemble target request
	targetRequest := bolt.NewRpcRequest(0, headers, buf)
	return targetRequest, buf, trailers, nil
}

func (t *http2bolt) TranscodingResponse(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) (types.HeaderMap, types.IoBuffer, types.HeaderMap, error) {
	sourceResponse, ok := headers.(*bolt.Response)
	if !ok {
		// if the response is not bolt response, it maybe come from hijack or send directly response.
		// so we just returns the original data
		return headers, buf, trailers, nil
	}
	targetResponse := fasthttp.Response{}

	// 1. headers
	sourceResponse.Range(func(Key, Value string) bool {
		targetResponse.Header.Set(Key, Value)
		return true
	})

	// 2. status code
	if sourceResponse.ResponseStatus != bolt.ResponseStatusSuccess {
		targetResponse.SetStatusCode(http.InternalServerError)
	}

	return http.ResponseHeader{ResponseHeader: &targetResponse.Header}, buf, trailers, nil
}
