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

package grpc

import (
	"context"
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/types"
)

type mockFilter struct{}

func (m mockFilter) OnDestroy() {}

func (m mockFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	return api.StreamFilterContinue
}

func (m mockFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {}

func TestStreamFilter(t *testing.T) {
	sfc := streamfilter.GetDefaultStreamFilterChain()
	ss := &grpcStreamFilterChain{
		DefaultStreamFilterChainImpl: sfc,
		phase:                        types.InitPhase,
		err:                          nil,
	}
	defer ss.destroy()
	ss.AddStreamReceiverFilter(&mockFilter{}, api.AfterRoute)
	status := ss.RunReceiverFilter(context.TODO(), api.AfterRoute, nil, nil, nil, ss.receiverFilterStatusHandler)
	if status != api.StreamFilterContinue {
		t.Fatalf("TestStreamFilter status: %v not equals with %v", status, api.StreamFilterContinue)
	}
}
