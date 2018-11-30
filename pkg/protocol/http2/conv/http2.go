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
	"github.com/alipay/sofa-mosn/pkg/protocol/mhttp2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	// TODO: remove while h2 branch is merged
	protocol.RegisterCommonConv(protocol.HTTP2, &common2httpOld{}, &http2commonOld{})
	protocol.RegisterCommonConv(protocol.MHTTP2, &common2httpOld{}, &http2common{})
}

// common -> http2 converter
type common2httpOld struct{}

func (c *common2httpOld) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
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

func (c *common2httpOld) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *common2httpOld) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

// http2 -> common converter
type http2commonOld struct{}

func (c *http2commonOld) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	if header, ok := headerMap.(protocol.CommonHeader); ok {

		cheader := make(map[string]string, len(header))

		// copy headers
		for k, v := range header {
			cheader[strings.ToLower(k)] = v
		}

		if _, ok := header[types.HeaderStatus]; ok {
			cheader[protocol.MosnHeaderDirection] = protocol.Response
		} else {
			cheader[protocol.MosnHeaderDirection] = protocol.Request
		}

		return protocol.CommonHeader(cheader), nil
	}
	return nil, errors.New("header type not supported")
}

func (c *http2commonOld) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *http2commonOld) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

// http2 -> common converter
type http2common struct{}

func (c *http2common) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	headers := mhttp2.DecodeHeader(headerMap)
	direction := ""
	switch  headerMap.(type) {
	case *mhttp2.ReqHeader:
		direction = protocol.Request
	case *mhttp2.RspHeader:
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
