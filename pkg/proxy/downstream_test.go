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

package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/trace"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestDownstream_FinishTracing_NotEnable(t *testing.T) {
	ds := downStream{context: context.Background()}
	ds.finishTracing()
	span := trace.SpanFromContext(context.Background())
	if span != nil {
		t.Error("Span is not nil")
	}
}

func TestDownstream_FinishTracing_Enable(t *testing.T) {
	trace.EnableTracing()
	ds := downStream{context: context.Background()}
	ds.finishTracing()
	span := trace.SpanFromContext(context.Background())
	if span != nil {
		t.Error("Span is not nil")
	}
}

func TestDownstream_FinishTracing_Enable_SpanIsNotNil(t *testing.T) {
	trace.EnableTracing()
	tracer := trace.CreateTracer("SOFATracer")
	trace.SetTracer(tracer)
	span := trace.Tracer().Start(time.Now())
	ctx := context.WithValue(context.Background(), trace.ActiveSpanKey, span)
	requestInfo := &network.RequestInfo{}
	ds := downStream{context: ctx, requestInfo: requestInfo}
	ds.finishTracing()

	span = trace.SpanFromContext(ctx)
	if span == nil {
		t.Error("Span is nil")
	}
	sofaTracerSpan := span.(*trace.SofaTracerSpan)
	zeroTime := time.Time{}
	if sofaTracerSpan.EndTime() == zeroTime {
		t.Error("Span is not finish")
	}
}

func TestDirectResponse(t *testing.T) {
	testCases := []struct {
		client *mockResponseSender
		route  *mockRoute
		check  func(t *testing.T, sender *mockResponseSender)
	}{
		// without body
		{
			client: &mockResponseSender{},
			route: &mockRoute{
				direct: &mockDirectRule{
					status: 500,
				},
			},
			check: func(t *testing.T, client *mockResponseSender) {
				if client.headers == nil {
					t.Fatal("want to receive a header response")
				}
				if code, ok := client.headers.Get(types.HeaderStatus); !ok || code != "500" {
					t.Error("response status code not expected")
				}
			},
		},
		// with body
		{
			client: &mockResponseSender{},
			route: &mockRoute{
				direct: &mockDirectRule{
					status: 400,
					body:   "mock 400 response",
				},
			},
			check: func(t *testing.T, client *mockResponseSender) {
				if client.headers == nil {
					t.Fatal("want to receive a header response")
				}
				if code, ok := client.headers.Get(types.HeaderStatus); !ok || code != "400" {
					t.Error("response status code not expected")
				}
				if client.data == nil {
					t.Fatal("want to receive a body response")
				}
				if client.data.String() != "mock 400 response" {
					t.Error("response  data not expected")
				}
			},
		},
	}
	for _, tc := range testCases {
		s := &downStream{
			proxy: &proxy{
				config: &v2.Proxy{},
				routersWrapper: &mockRouterWrapper{
					routers: &mockRouters{
						route: tc.route,
					},
				},
				clusterManager: &mockClusterManager{},
				readCallbacks:  &mockReadFilterCallbacks{},
				stats:          globalStats,
				listenerStats:  newListenerStats("test"),
			},
			logger:         log.DefaultLogger,
			responseSender: tc.client,
			requestInfo:    &network.RequestInfo{},
		}
		// event call Receive Headers
		// trigger direct response
		s.matchRoute()
		s.ReceiveHeaders(nil, false)
		// check
		tc.check(t, tc.client)
	}
}
