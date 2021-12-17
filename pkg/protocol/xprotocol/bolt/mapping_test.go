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

package bolt

import (
	"context"
	"testing"

	"mosn.io/api"
	"mosn.io/pkg/header"
)

func TestSofaMapping(t *testing.T) {
	m := BoltStatusMapping{}
	testcases := []struct {
		Header   api.HeaderMap
		Expected int
	}{
		{
			Header:   NewRpcResponse(0, ResponseStatusSuccess, nil, nil),
			Expected: 200,
		},
		{
			Header:   NewRpcResponse(0, ResponseStatusServerThreadpoolBusy, nil, nil),
			Expected: 503,
		},
		{
			Header:   NewRpcResponse(0, ResponseStatusTimeout, nil, nil),
			Expected: 504,
		},
		{
			Header:   NewRpcResponse(0, ResponseStatusClientSendError, nil, nil),
			Expected: 500,
		},
		{
			Header:   NewRpcResponse(0, ResponseStatusConnectionClosed, nil, nil),
			Expected: 502,
		},
		{
			Header:   NewRpcResponse(0, ResponseStatusError, nil, nil),
			Expected: 500,
		},
		{
			Header:   header.CommonHeader{},
			Expected: 0,
		},
	}
	for i, tc := range testcases {
		code, _ := m.MappingHeaderStatusCode(context.Background(), tc.Header)
		if code != tc.Expected {
			t.Errorf("#%d get unexpected code", i)
		}

	}
}
