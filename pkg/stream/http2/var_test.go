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
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

func Test_get_prefixProtocolVar(t *testing.T) {
	headerName := "Header_key"
	expect := "header_value"
	headers := http2.NewHeaderMap(http.Header(map[string][]string{}))
	headers.Set(headerName, expect)

	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamReqHeaders, headers)
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, protocol.HTTP2)

	actual, err := variable.GetProtocolResource(ctx, api.HEADER, headerName)
	assert.NoErrorf(t, err, "get protocol header failed")
	assert.Equalf(t, expect, actual, "header value expect to be (%s), but get (%s)", expect, actual)

	errHeader := "err_test"
	actual, err = variable.GetProtocolResource(ctx, api.HEADER, errHeader)
	assert.Errorf(t, err, "get protocol header err test failed")
	assert.Equalf(t, variable.ErrValueNotFound.Error(), err.Error(), "header error test expect to be (%s), but get (%s)", variable.ErrValueNotFound.Error(), actual)

	// test cookie
	cookieName := "cookie_key"
	expect = "cookie_value"
	headers.Set("Cookie", "cookie_key=cookie_value; fake_cookie_key=fake_cookie_value;")
	actual, err = variable.GetProtocolResource(ctx, api.COOKIE, cookieName)
	assert.NoErrorf(t, err, "get protocol cookie failed")
	assert.Equalf(t, expect, actual, "cookie value expect to be (%s), but get (%s)", expect, actual)

	errCookie := "err_test"
	actual, err = variable.GetProtocolResource(ctx, api.COOKIE, errCookie)
	assert.Errorf(t, err, "get protocol cookie err test failed")
	assert.Equalf(t, variable.ErrValueNotFound.Error(), err.Error(), "cookie error test expect to be (%s), but get (%s)", variable.ErrValueNotFound.Error(), actual)

}

func Test_get_scheme(t *testing.T) {
	expect := "https"
	ctx := variable.NewVariableContext(context.Background())

	_ = variable.Set(ctx, types.VariableDownStreamProtocol, protocol.HTTP2)

	variable.SetString(ctx, types.VarScheme, expect)
	actual, err := variable.GetProtocolResource(ctx, api.SCHEME)
	assert.NoErrorf(t, err, variable.ErrValueNotFound.Error())
	assert.Equalf(t, expect, actual, "header value expect to be (%s), but get (%s)", expect, actual)
}
