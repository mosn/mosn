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

package sofarpc

import (
	"testing"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestSofaMapping(t *testing.T) {
	m := &sofaMapping{}
	testcases := []struct {
		Header   types.HeaderMap
		Expected int
	}{
		{
			Header: &BoltResponseCommand{
				ResponseStatus: RESPONSE_STATUS_SUCCESS,
			},
			Expected: 200,
		},
		{
			Header: &BoltResponseCommand{
				ResponseStatus: RESPONSE_STATUS_SERVER_THREADPOOL_BUSY,
			},
			Expected: 503,
		},
		{
			Header: &BoltResponseCommand{
				ResponseStatus: RESPONSE_STATUS_TIMEOUT,
			},
			Expected: 504,
		},
		{
			Header: &BoltResponseCommand{
				ResponseStatus: RESPONSE_STATUS_CLIENT_SEND_ERROR,
			},
			Expected: 500,
		},
		{
			Header: &BoltResponseCommand{
				ResponseStatus: RESPONSE_STATUS_CONNECTION_CLOSED,
			},
			Expected: 502,
		},
		{
			Header: &BoltResponseCommand{
				ResponseStatus: RESPONSE_STATUS_ERROR,
			},
			Expected: 500,
		},
		{
			Header:   protocol.CommonHeader{},
			Expected: 0,
		},
	}
	for i, tc := range testcases {
		code, _ := m.MappingHeaderStatusCode(tc.Header)
		if code != tc.Expected {
			t.Errorf("#%d get unexpected code", i)
		}

	}
}
