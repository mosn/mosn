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
	"strings"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/http"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/valyala/fasthttp"
)

func init() {
	protocol.RegisterCommonConv(protocol.HTTP1, &common2http{}, &http2common{})
}

// common -> http converter
type common2http struct{}

func (c *common2http) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	if header, ok := headerMap.(protocol.CommonHeader); ok {
		direction, ok := header[protocol.MosnHeaderDirection]
		if !ok {
			return nil, protocol.ErrHeaderDirection
		}

		delete(header, protocol.MosnHeaderDirection)

		switch direction {
		case protocol.Request:
			headerImpl := &fasthttp.RequestHeader{}
			// copy headers
			for k, v := range header {
				headerImpl.Set(k, v)
			}
			return http.RequestHeader{headerImpl}, nil
		case protocol.Response:
			headerImpl := &fasthttp.ResponseHeader{}
			// copy headers
			for k, v := range header {
				headerImpl.Set(k, v)
			}
			return http.ResponseHeader{headerImpl}, nil
		}
	}
	return nil, errors.New("header type not supported")
}

func (c *common2http) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *common2http) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

// http -> common converter
type http2common struct{}

func (c *http2common) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	switch header := headerMap.(type) {
	case http.RequestHeader:
		cheader := make(map[string]string, header.Len())

		// copy headers
		header.VisitAll(func(key, value []byte) {
			cheader[strings.ToLower(string(key))] = string(value)
		})

		cheader[protocol.MosnHeaderDirection] = protocol.Request

		return protocol.CommonHeader(cheader), nil
	case http.ResponseHeader:
		cheader := make(map[string]string, header.Len())

		// copy headers
		header.VisitAll(func(key, value []byte) {
			cheader[strings.ToLower(string(key))] = string(value)
		})

		cheader[protocol.MosnHeaderDirection] = protocol.Response

		return protocol.CommonHeader(cheader), nil
	}
	return nil, errors.New("header type not supported")
}

func (c *http2common) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *http2common) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}
