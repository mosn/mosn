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
	"testing"

	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestMapping(t *testing.T) {
	if _, err := MappingHeaderStatusCode(SofaRPC, nil); err != ErrNoMapping {
		t.Error("no register type")
	}
	testcases := []struct {
		Header   types.HeaderMap
		Expetced int
	}{
		{
			CommonHeader{types.HeaderStatus: "200"},
			200,
		},
		{
			CommonHeader{},
			0,
		},
	}
	for i, tc := range testcases {
		code, _ := MappingHeaderStatusCode(HTTP1, tc.Header)
		if code != tc.Expetced {
			t.Errorf("#%d unexpected status code", i)
		}
	}

}
