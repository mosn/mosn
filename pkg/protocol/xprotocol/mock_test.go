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

package xprotocol

import (
	"context"
	"sync/atomic"

	"mosn.io/api"

	"mosn.io/mosn/pkg/types"
)

var mockProtocolName = types.ProtocolName("mock-protocol")

type mockProtocol struct {
}

func (mp *mockProtocol) Name() types.ProtocolName {
	return mockProtocolName
}

func (mp *mockProtocol) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	return nil, nil
}

func (mp *mockProtocol) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	return nil, nil
}

// Heartbeater
func (mp *mockProtocol) Trigger(ctx context.Context, requestId uint64) api.XFrame {
	return nil
}

func (mp *mockProtocol) Reply(ctx context.Context, request api.XFrame) api.XRespFrame {
	return nil
}

// Hijacker
func (mp *mockProtocol) Hijack(ctx context.Context, request api.XFrame, statusCode uint32) api.XRespFrame {
	return nil
}

func (mp *mockProtocol) Mapping(httpStatusCode uint32) uint32 {
	return 0
}

func (mp *mockProtocol) PoolMode() api.PoolMode {
	return api.Multiplex
}

func (mp *mockProtocol) EnableWorkerPool() bool {
	return true
}

func (mp *mockProtocol) GenerateRequestID(streamID *uint64) uint64 {
	return atomic.AddUint64(streamID, 1)
}

func mockMatcher(data []byte) api.MatchResult {
	return api.MatchSuccess
}

type mockMapping struct {
}

func (mm *mockMapping) MappingHeaderStatusCode(ctx context.Context, headers types.HeaderMap) (int, error) {
	return 200, nil
}
