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

package variable

import (
	"context"
	"testing"

	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
)

const (
	HTTP1 types.ProtocolName = "Http1"
	Dubbo types.ProtocolName = "Dubbo"
)

func TestGetProtocolResource(t *testing.T) {
	m := make(map[string]string)
	httpKey := string(HTTP1) + "_" + types.VarProtocolRequestPath
	dubboKey := string(Dubbo) + "_" + types.VarProtocolRequestPath
	m[httpKey] = "/http"
	m[dubboKey] = "/dubbo"

	for k, _ := range m {
		val := m[k]
		// register test variable
		RegisterVariable(NewBasicVariable(k, nil, func(ctx context.Context, variableValue *IndexedValue, data interface{}) (s string, err error) {
			return val, nil
		}, nil, 0))
	}

	// register HTTP protocol resource var
	RegisterProtocolResource(HTTP1, api.PATH, types.VarProtocolRequestPath)

	ctx := context.Background()

	ctx = mosnctx.WithValue(ctx, types.ContextKeyDownStreamProtocol, HTTP1)
	vv, err := GetProtocolResource(ctx, api.PATH)
	if err != nil {
		t.Error(err)
	}

	if vv != m[httpKey] {
		t.Errorf("get value not equal, expected: %s, acutal: %s", m[httpKey], vv)
	}

	ctx = mosnctx.WithValue(ctx, types.ContextKeyDownStreamProtocol, Dubbo)
	vv, err = GetProtocolResource(ctx, api.PATH)
	if err.Error() != errUnregisterProtocolResource+string(Dubbo) {
		t.Fatal("unexpected get error")
	}
}

func BenchmarkGetProtocolResource(b *testing.B) {
	ctx := prepareProtocolResource()
	for i := 0; i < b.N; i++ {
		_, err := GetProtocolResource(ctx, api.PATH)
		if err != nil {
			b.Error("get variable failed:", err)
		}
	}
}

func BenchmarkGetVariableValue(b *testing.B) {

	ctx := prepareProtocolResource()
	for i := 0; i < b.N; i++ {
		_, err := GetVariableValue(ctx, string(api.PATH))
		if err != nil {
			b.Error("get variable failed:", err)
		}
	}
}

func prepareProtocolResource() context.Context {
	name := "http_request_path"
	value := "/path"
	// register test variable
	RegisterVariable(NewBasicVariable(name, nil, func(ctx context.Context, variableValue *IndexedValue, data interface{}) (s string, err error) {
		return value, nil
	}, nil, 0))

	// register HTTP protocol resource var
	RegisterProtocolResource(HTTP1, api.PATH, name)

	ctx := context.Background()
	ctx = mosnctx.WithValue(ctx, types.ContextKeyDownStreamProtocol, HTTP1)
	return ctx
}
