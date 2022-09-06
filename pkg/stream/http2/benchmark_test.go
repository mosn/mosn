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

package http2

import (
	"context"
	"net/http"
	"testing"

	"mosn.io/api"
	"mosn.io/pkg/variable"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
)

func BenchmarkGetPrefixProtocolVarCookie(b *testing.B) {
	cookieName := "cookie_key"
	headers := http2.NewHeaderMap(http.Header(map[string][]string{}))
	headers.Set("Cookie", "cookie_key=cookie_value; fake_cookie_key=fake_cookie_value;")

	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamReqHeaders, headers)
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, protocol.HTTP2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = variable.GetProtocolResource(ctx, api.COOKIE, cookieName)
	}
}

func BenchmarkGetPrefixProtocolVarHeader(b *testing.B) {
	headerName := "Header_key"
	expect := "header_value"
	headers := http2.NewHeaderMap(http.Header(map[string][]string{}))
	headers.Set(headerName, expect)

	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamReqHeaders, headers)
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, protocol.HTTP2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = variable.GetProtocolResource(ctx, api.HEADER, headerName)
	}
}
