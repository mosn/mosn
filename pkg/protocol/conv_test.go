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

package protocol

import (
	"context"
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

type common2mock struct{}
type mock2common struct{}

const mockProtocol = "mock"

func (c *common2mock) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return nil, nil
}

func (c *common2mock) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *common2mock) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

func (c *mock2common) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return nil, nil
}

func (c *mock2common) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *mock2common) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

func TestConvertHeader(t *testing.T) {
	RegisterCommonConv(mockProtocol, &common2mock{}, &mock2common{})
	testcases := []struct {
		src    api.Protocol
		dst    api.Protocol
		header api.HeaderMap
		data   buffer.IoBuffer
	}{
		{
			common,
			mockProtocol,
			CommonHeader{types.HeaderStatus: "200"},
			nil,
		},
		{
			common,
			mockProtocol,
			CommonHeader{types.HeaderStatus: "200"},
			nil,
		},
	}
	for i, tc := range testcases {
		_, err := ConvertHeader(context.Background(), tc.src, tc.dst, tc.header)
		if err != nil {
			t.Errorf("#%d case ConvertHeader failed: %v", i, err)
		}

		_, err = ConvertData(context.Background(), tc.src, tc.dst, tc.data)
		if err != nil {
			t.Errorf("#%d case ConvertData failed: %v", i, err)
		}

		_, err = ConvertTrailer(context.Background(), tc.src, tc.dst, nil)
		if err != nil {
			t.Errorf("#%d case ConvertTrailer failed: %v", i, err)
		}
	}

	// test invalid
	testcases = []struct {
		src    api.Protocol
		dst    api.Protocol
		header api.HeaderMap
		data   buffer.IoBuffer
	}{
		{
			"unknown",
			mockProtocol,
			CommonHeader{types.HeaderStatus: "200"},
			nil,
		},
	}
	for i, tc := range testcases {
		_, err := ConvertHeader(context.Background(), tc.src, tc.dst, tc.header)
		if err == nil {
			t.Errorf("#%d case ConvertHeader failed", i)
		}

		_, err = ConvertData(context.Background(), tc.src, tc.dst, tc.data)
		if err == nil {
			t.Errorf("#%d case ConvertData failed", i)
		}

		_, err = ConvertTrailer(context.Background(), tc.src, tc.dst, nil)
		if err == nil {
			t.Errorf("#%d case ConvertTrailer failed", i)
		}
	}

}
