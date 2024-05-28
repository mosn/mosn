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
	"fmt"
	"strings"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

var (
	headerIndex = len(types.VarPrefixHttp2Header)
	cookieIndex = len(types.VarPrefixHttp2Cookie)

	builtinVariables = []variable.Variable{
		variable.NewStringVariable(types.VarHttp2RequestScheme, nil, schemeGetter, nil, 0),
		variable.NewVariable(types.VarHttp2RequestUseStream, nil, nil, variable.DefaultSetter, 0),
		variable.NewVariable(types.VarHttp2ResponseUseStream, nil, nil, variable.DefaultSetter, 0),
	}

	prefixVariables = []variable.Variable{
		variable.NewStringVariable(types.VarPrefixHttp2Header, nil, headerGetter, nil, 0),
		variable.NewStringVariable(types.VarPrefixHttp2Cookie, nil, cookieGetter, nil, 0),
	}
)

func init() {
	// register built-in variables
	for idx := range builtinVariables {
		variable.Register(builtinVariables[idx])
	}

	// register prefix variables, like header_xxx/arg_xxx/cookie_xxx
	for idx := range prefixVariables {
		variable.RegisterPrefix(prefixVariables[idx].Name(), prefixVariables[idx])
	}

	// register protocol resource
	variable.RegisterProtocolResource(protocol.HTTP2, api.SCHEME, types.VarProtocolRequestScheme)
	variable.RegisterProtocolResource(protocol.HTTP2, api.HEADER, types.VarProtocolRequestHeader)
	variable.RegisterProtocolResource(protocol.HTTP2, api.COOKIE, types.VarProtocolCookie)
}

func schemeGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	scheme, err := variable.GetString(ctx, types.VarScheme)
	if err != nil || scheme == "" {
		return variable.ValueNotFound, variable.ErrValueNotFound
	}
	return scheme, nil
}

func headerGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (s string, err error) {
	hv, err := variable.Get(ctx, types.VariableDownStreamReqHeaders)
	if err != nil {
		return variable.ValueNotFound, variable.ErrValueNotFound
	}
	headers, ok := hv.(api.HeaderMap)
	if !ok {
		return variable.ValueNotFound, variable.ErrValueNotFound
	}
	headerKey, ok := data.(string)
	if !ok {
		return variable.ValueNotFound, variable.ErrValueNotFound
	}

	header, found := headers.Get(headerKey[headerIndex:])
	if !found {
		return variable.ValueNotFound, variable.ErrValueNotFound
	}

	return header, nil
}

func cookieGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (s string, err error) {
	hv, err := variable.Get(ctx, types.VariableDownStreamReqHeaders)
	if err != nil {
		return variable.ValueNotFound, variable.ErrValueNotFound
	}
	headers, ok := hv.(api.HeaderMap)
	if !ok {
		return variable.ValueNotFound, variable.ErrValueNotFound
	}
	cookieKey, ok := data.(string)
	if !ok {
		return variable.ValueNotFound, variable.ErrValueNotFound
	}

	cookiePrefix := fmt.Sprintf("%s=", cookieKey[cookieIndex:])
	cookieString, found := headers.Get("Cookie")
	if !found {
		return variable.ValueNotFound, variable.ErrValueNotFound
	}

	for _, cookieKV := range strings.Split(cookieString, ";") {
		kv := strings.TrimSpace(cookieKV)
		if strings.HasPrefix(kv, cookiePrefix) {
			return strings.TrimSpace(strings.TrimPrefix(kv, cookiePrefix)), nil
		}
	}
	return variable.ValueNotFound, variable.ErrValueNotFound
}
