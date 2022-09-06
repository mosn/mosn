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

package router

import (
	"context"
	"net"
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

func BenchmarkStringGenerateHash(b *testing.B) {
	testProtocol := types.ProtocolName("SomeProtocol")
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, testProtocol)

	headerGetter := func(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
		return "test_header_value", nil
	}
	headerValue := variable.NewStringVariable("SomeProtocol_request_header_", nil, headerGetter, nil, 0)
	variable.RegisterPrefix(headerValue.Name(), headerValue)
	variable.RegisterProtocolResource(testProtocol, api.HEADER, types.VarProtocolRequestHeader)
	headerHp := headerHashPolicyImpl{
		key: "header_key",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = headerHp.GenerateHash(ctx)
	}
}

func BenchmarkIPGenerateHash(b *testing.B) {
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableOriRemoteAddr, &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 80,
	})
	sourceIPHp := sourceIPHashPolicyImpl{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = sourceIPHp.GenerateHash(ctx)
	}
}
