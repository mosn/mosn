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
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/proxy"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

type mockReceiveHandler struct {
	api.StreamReceiverFilterHandler
	phase types.Phase
}

func (f *mockReceiveHandler) RequestInfo() api.RequestInfo {
	return nil
}

func (f *mockReceiveHandler) GetFilterCurrentPhase() api.ReceiverFilterPhase {
	// default AfterRoute
	p := api.AfterRoute

	switch f.phase {
	case types.DownFilter:
		p = api.BeforeRoute
	case types.DownFilterAfterRoute:
		p = api.AfterRoute
	case types.DownFilterAfterChooseHost:
		p = api.AfterChooseHost
	}

	return p
}

func TestDSLStreamFilter(t *testing.T) {
	cfg := v2.StreamDSL{
		BeforeRouterDSL:  "conditional((request.host == \"dsl\") && (request.headers[\"dsl\"] == \"dsl\"), ctx.rewrite_request_url(string(0)), ctx.rewrite_request_url(\"/xxx0\"))",
		AfterRouterDSL:   "conditional((request.host == \"dsl\") && (request.headers[\"dsl\"] == \"dsl\"), ctx.rewrite_request_url(\"1\"), ctx.rewrite_request_url(\"/xxx1\"))",
		AfterBalancerDSL: "conditional((request.host == \"dsl\") && (request.headers[\"dsl\"] == \"dsl\"), ctx.rewrite_request_url(\"2\"), ctx.rewrite_request_url(\"/xxx2\"))",
		SendFilterDSL:    "conditional(response.headers[\"dsl0\"] == \"dsl0\", ctx.add_response_header(\"dsl1\", \"dsl1\"), ctx.add_response_header(\"dsl2\", \"dsl2\"))",
		LogDSL:           "conditional(response.headers[\"dsl0\"] == \"dsl0\", ctx.del_response_header(\"dsl0\"), ctx.del_response_header(\"dsl1\"))",
	}

	dsl, err := checkDSL(cfg)
	if err != nil {
		t.Errorf("check dsl failed: %v", err)
	}

	f := NewDSLFilter(context.Background(), dsl)
	if f == nil {
		t.Error("create dsl filter failed!")
	}

	receiveHandler := &mockReceiveHandler{}

	f.SetReceiveFilterHandler(receiveHandler)
	reqHeaders := protocol.CommonHeader(map[string]string{
		"dsl": "dsl",
	})

	respHeaders := protocol.CommonHeader(map[string]string{
		"dsl0": "dsl0",
	})

	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamReqHeaders, reqHeaders)
	phase := []types.Phase{types.DownFilter, types.DownFilterAfterRoute, types.DownFilterAfterChooseHost}
	variable.SetString(ctx, types.VarPath, "/dsl")
	variable.SetString(ctx, types.VarHost, "dsl")
	for k, p := range phase {
		receiveHandler.phase = p
		f.OnReceive(ctx, reqHeaders, nil, nil)
		// should fetch rewrite path from ctx
		if v, err := variable.GetString(ctx, types.VarPath); err != nil || v != strconv.Itoa(k) {
			t.Errorf("DSL execute failed, index: %d, want: %s but: %s", k, strconv.Itoa(k), v)
		}

	}

	// add dsl1 respheader
	_ = variable.Set(ctx, types.VariableDownStreamRespHeaders, respHeaders)
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

func TestNewDSLStreamFilter(t *testing.T) {
	m := map[string]interface{}{}
	if _, err := CreateDSLFilterFactory(m); err != nil {
		t.Errorf("Create DSL filter failed: %v", err)
	}

	// test check dsl
	m["before_router_by_dsl"] = "conditional((request.host == \"dsl\") && (request.headers[\"dsl\"] == \"dsl\"), ctx.rewrite_request_url(string(0)), ctx.rewrite_request_url(\"/xxx0\"))"
	cf, err := CreateDSLFilterFactory(m)
	if cf == nil || err != nil {
		t.Errorf("Create DSL filter cf failed: %v", err)
	}
	// test panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("CreateFilterChain should be panic when callback is nil!")
		}
	}()

	cf.CreateFilterChain(nil, nil)

	// test unknown function or variables
	m["before_router_by_dsl"] = "conditional((request.unknown == \"dsl\"), unknown_function(string(0)), ctx.rewrite_request_url(\"/xxx0\"))"
	if _, err := CreateDSLFilterFactory(m); err == nil {
		t.Errorf("It should be failed when use invalid dsl.")
	}

	m["log_filter_by_dsl"] = "conditional((request.unknown == \"dsl\"), unknown_function(string(0)), ctx.rewrite_request_url(\"/xxx0\"))"
	if _, err := CreateDSLFilterFactory(m); err == nil {
		t.Errorf("It should be failed when use invalid dsl.")
	}

	// test unknown phase
	m["unknown_by_dsl"] = "conditional((request.host == \"dsl\")), ctx.rewrite_request_url(string(0)), ctx.rewrite_request_url(\"xxx\"))"
	if _, err := CreateDSLFilterFactory(m); err == nil {
		t.Errorf("It should be failed when use invalid phase.")
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
		"dsl": "dsl",
	})

	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableDownStreamReqHeaders, reqHeaders)
	variable.SetString(ctx, types.VarPath, "/dsl")
	variable.SetString(ctx, types.VarHost, "dsl")
	receiveHandler.phase = types.DownFilter
	want := "0"
	for i := 0; i < b.N; i++ {
		f.OnReceive(ctx, reqHeaders, nil, nil)
		if v, err := variable.GetString(ctx, types.VarPath); err != nil || v != want {
			b.Errorf("DSL execute failed, want: %s but: %s", want, v)
		}
	}
}
