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

package proxy

import (
	"context"
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

func TestResponseFlag(t *testing.T) {
	testCases := []struct {
		name         string
		responseFlag api.ResponseFlag
		want         string
	}{
		{
			name:         "not-set-any-flags",
			responseFlag: api.ResponseFlag(0),
			want:         "false",
		}, {
			name:         "set-one-flag",
			responseFlag: api.NoHealthyUpstream,
			want:         "true",
		}, {
			name:         "set-multiple-flags",
			responseFlag: api.NoHealthyUpstream | api.UpstreamRequestTimeout,
			want:         "true",
		},
	}

	for _, tc := range testCases {
		var ctx context.Context
		ctx = buffer.NewBufferPoolContext(ctx)
		pbuf := proxyBuffersByContext(ctx)
		pbuf.info.SetResponseFlag(tc.responseFlag)

		varFlag := variable.NewStringVariable(types.VarResponseFlag, nil, responseFlagGetter, nil, 0)
		val, err := varFlag.Getter().Get(ctx, nil, nil)
		if err != nil {
			t.Fatalf("%s: failed to get value of response_flag, err:%s", tc.name, err)
		}
		if tc.want != val.(string) {
			t.Errorf("%s: response flag expected (%v), but got (%v)", tc.name, tc.want, val)
		}
	}
}

func TestResponseHeaderNotFound(t *testing.T) {
	variable.NewStringVariable(types.VarPrefixRespHeader, nil, responseHeaderMapGetter, nil, 0)

	var ctx context.Context
	ctx = buffer.NewBufferPoolContext(ctx)
	val, err := variable.GetString(ctx, "response_header_not_exists")
	if val != "" || err == nil {
		t.Fatalf("unexpected variable value, expect not found while got: %v", val)
	}
}

func TestRequestHeaderNotFound(t *testing.T) {
	variable.NewStringVariable(types.VarPrefixReqHeader, nil, requestHeaderMapGetter, nil, 0)

	var ctx context.Context
	ctx = buffer.NewBufferPoolContext(ctx)
	val, err := variable.GetString(ctx, "request_header_not_exists")
	if val != "" || err == nil {
		t.Fatalf("unexpected variable value, expect not found while got: %v", val)
	}
}

func TestReturnError(t *testing.T) {

	type testFunc func(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error)
	testCases := []struct {
		name     string
		testFunc testFunc
		want     string
	}{
		{
			name:     "upstreamLocalAddressGetter",
			testFunc: upstreamLocalAddressGetter,
		},
		{
			name:     "downstreamLocalAddressGetter",
			testFunc: downstreamLocalAddressGetter,
		},
		{
			name:     "downstreamRemoteAddressGetter",
			testFunc: downstreamRemoteAddressGetter,
		},
		{
			name:     "upstreamHostGetter",
			testFunc: upstreamHostGetter,
		},
		{
			name:     "upstreamClusterGetter",
			testFunc: upstreamClusterGetter,
		},
		{
			name:     "requestHeaderMapGetter",
			testFunc: requestHeaderMapGetter,
		},
		{
			name:     "responseHeaderMapGetter",
			testFunc: responseHeaderMapGetter,
		},
	}

	var ctx context.Context
	ctx = buffer.NewBufferPoolContext(ctx)

	for _, tc := range testCases {
		var ctx context.Context
		ctx = buffer.NewBufferPoolContext(ctx)

		_, err := tc.testFunc(ctx, nil, nil)

		if err.Error() != variable.ErrValueNotFound.Error() {
			t.Errorf("%s: response flag expected (%s), but got (%s)", tc.name, tc.want, err.Error())
		}
	}
}
