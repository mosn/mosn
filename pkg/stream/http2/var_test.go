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

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

func Test_get_prefixProtocolVar(t *testing.T) {
	headerName := "Header_key"
	expect := "header_value"
	headers := http2.NewHeaderMap(http.Header(map[string][]string{}))
	headers.Set(headerName, expect)

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyDownStreamHeaders, headers)

	ctx = mosnctx.WithValue(ctx, types.ContextKeyDownStreamProtocol, protocol.HTTP2)

	actual, err := variable.GetProtocolResource(ctx, api.HEADER, headerName)
	assert.NoErrorf(t, err, "get protocol header failed")
	assert.Equalf(t, expect, actual, "header value expect to be %s, but get %s")

	// test cookie
	cookieName := "cookie_key"
	expect = "cookie_value"
	headers.Set("Cookie", "cookie_key=cookie_value; fake_cookie_key=fake_cookie_value;")
	actual, err = variable.GetProtocolResource(ctx, api.COOKIE, cookieName)
	assert.NoErrorf(t, err, "get protocol cookie failed")
	assert.Equalf(t, expect, actual, "cookie value expect to be %s, but get %s")
}
