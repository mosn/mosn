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

package conv

import (
	"context"
	"errors"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
)

func init() {
	protocol.RegisterCommonConv(protocol.HTTP2, &common2http{}, &http2common{})
}

// common -> http2 converter
type common2http struct{}

func (c *common2http) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	if header, ok := headerMap.(protocol.CommonHeader); ok {
		cheader := make(map[string]string, len(header))

		delete(header, protocol.MosnHeaderDirection)

		// copy headers
		for k, v := range header {
			cheader[k] = v
		}

		return protocol.CommonHeader(cheader), nil
	}
	return nil, errors.New("header type not supported")
}

func (c *common2http) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *common2http) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

// http2 -> common converter
type http2common struct{}

func (c *http2common) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	headers := http2.DecodeHeader(headerMap)
	direction := ""
	switch headerMap.(type) {
	case *http2.ReqHeader:
		direction = protocol.Request
	case *http2.RspHeader:
		direction = protocol.Response
	default:
		return nil, errors.New("header type not supported")
	}

	headers.Set(protocol.MosnHeaderDirection, direction)
	return headers, nil
}

func (c *http2common) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *http2common) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}
