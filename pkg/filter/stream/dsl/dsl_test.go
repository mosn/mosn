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

package dsl

import (
	"context"
	"strconv"
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/proxy"
	"mosn.io/mosn/pkg/types"
)

type mockReceiveHandler struct {
	api.StreamReceiverFilterHandler
}

func (f *mockReceiveHandler) RequestInfo() api.RequestInfo {
	return nil
}

func TestDSLNewStreamFilter(t *testing.T) {
	cfg := v2.StreamDSL{
		BeforeRouterDSL:  "conditional((request.host == \"dsl\") && (request.headers[\"dsl\"] == \"dsl\"), ctx.rewrite_request_url(string(0)), ctx.rewrite_request_url(\"/xxx0\"))",
		AfterRouterDSL:   "conditional((request.host == \"dsl\") && (request.headers[\"dsl\"] == \"dsl\"), ctx.rewrite_request_url(\"1\"), ctx.rewrite_request_url(\"/xxx1\"))",
		AfterBalancerDSL: "conditional((request.host == \"dsl\") && (request.headers[\"dsl\"] == \"dsl\"), ctx.rewrite_request_url(\"2\"), ctx.rewrite_request_url(\"/xxx2\"))",
		SendFilterDSL:    "conditional(response.headers[\"dsl0\"] == \"dsl0\", ctx.add_response_header(\"dsl1\", \"dsl1\"), ctx.add_response_header(\"dsl2\", \"dsl2\"))",
		LogDSL:           "conditional(response.headers[\"dsl0\"] == \"dsl0\", ctx.del_response_header(\"dsl0\"), ctx.del_response_header(\"dsl1\"))",
	}

	dsl, err := checkDSL(cfg)
	if err != nil {
		t.Errorf("check dsl faied: %v", err)
	}

	f := NewDSLFilter(context.Background(), dsl)

	receiveHandler := &mockReceiveHandler{}

	f.SetReceiveFilterHandler(receiveHandler)
	reqHeaders := protocol.CommonHeader(map[string]string{
		protocol.MosnHeaderPathKey: "/dsl",
		protocol.MosnHeaderHostKey: "dsl",
		"dsl":                      "dsl",
	})

	respHeaders := protocol.CommonHeader(map[string]string{
		"dsl0": "dsl0",
	})

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyDownStreamHeaders, reqHeaders)
	phase := []types.Phase{types.DownFilter, types.DownFilterAfterRoute, types.DownFilterAfterChooseHost}
	for k, p := range phase {

		ctx = mosnctx.WithValue(ctx, types.ContextKeyStreamFilterPhase, p)
		f.OnReceive(ctx, reqHeaders, nil, nil)
		if v, ok := reqHeaders.Get(protocol.MosnHeaderPathKey); !ok || v != strconv.Itoa(k) {
			t.Errorf("DSL execute failed, index: %d, want: %s but: %s", k, strconv.Itoa(k), v)
		}

	}

	// add dsl1 respheader
	ctx = mosnctx.WithValue(ctx, types.ContextKeyDownStreamRespHeaders, respHeaders)
	f.Append(ctx, respHeaders, nil, nil)
	if _, ok := respHeaders.Get("dsl1"); !ok {
		t.Error("DSL execute failed, at the Append phase")
	}

	// delete dsl0 respheader
	f.Log(ctx, nil, respHeaders, nil)

	if _, ok := respHeaders.Get("dsl0"); ok {
		t.Error("DSL execute failed, at the Log phase")
	}

}

func BenchmarkDSL(b *testing.B) {
	cfg := v2.StreamDSL{
		BeforeRouterDSL: "conditional((request.host == \"dsl\") && (request.headers[\"dsl\"] == \"dsl\"), ctx.rewrite_request_url(string(0)), ctx.rewrite_request_url(\"/xxx0\"))",
	}

	dsl, err := checkDSL(cfg)
	if err != nil {
		b.Errorf("check dsl faied: %v", err)
	}

	f := NewDSLFilter(context.Background(), dsl)

	receiveHandler := &mockReceiveHandler{}

	f.SetReceiveFilterHandler(receiveHandler)
	reqHeaders := protocol.CommonHeader(map[string]string{
		protocol.MosnHeaderPathKey: "/dsl",
		protocol.MosnHeaderHostKey: "dsl",
		"dsl":                      "dsl",
	})

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyDownStreamHeaders, reqHeaders)

	ctx = mosnctx.WithValue(ctx, types.ContextKeyStreamFilterPhase, types.DownFilter)

	want := "0"
	for i := 0; i < b.N; i++ {
		f.OnReceive(ctx, reqHeaders, nil, nil)
		if v, ok := reqHeaders.Get(protocol.MosnHeaderPathKey); !ok || v != want {
			b.Errorf("DSL execute failed, want: %s but: %s", want, v)
		}
	}
}
