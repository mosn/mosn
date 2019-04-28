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

package http

import (
	"context"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	mosnctx "github.com/alipay/sofa-mosn/pkg/context"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/models"
	"github.com/alipay/sofa-mosn/pkg/trace"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/valyala/fasthttp"
)

var spanBuilder = &SpanBuilder{}

type SpanBuilder struct {
}

func (spanBuilder *SpanBuilder) BuildSpan(args ...interface{}) types.Span {
	if len(args) == 0 {
		return nil
	}

	ctx, ok := args[0].(context.Context)
	if !ok {
		log.Proxy.Errorf(ctx, "[stream] [http] span build failed, first arg unexpected:%+v", args[0])
		return nil
	}

	request, ok := args[1].(*fasthttp.Request)
	if !ok {
		log.Proxy.Errorf(ctx, "[stream] [http] span build failed, second arg unexpected:%+v", args[0])
		return nil
	}

	span := trace.Tracer().Start(time.Now())

	traceId := string(request.Header.Peek(models.TRACER_ID_KEY))
	if traceId == "" {
		traceId = trace.IdGen().GenerateTraceId()
	}
	span.SetTag(trace.TRACE_ID, traceId)
	lType := mosnctx.Get(ctx, types.ContextKeyListenerType)

	spanId := string(request.Header.Peek(models.RPC_ID_KEY))
	if spanId == "" {
		spanId = "0" // Generate a new span id
	} else {
		if lType == v2.INGRESS {
			trace.AddSpanIdGenerator(trace.NewSpanIdGenerator(traceId, spanId))
		} else if lType == v2.EGRESS {
			span.SetTag(trace.PARENT_SPAN_ID, spanId)
			spanKey := &trace.SpanKey{TraceId: traceId, SpanId: spanId}
			if spanIdGenerator := trace.GetSpanIdGenerator(spanKey); spanIdGenerator != nil {
				spanId = spanIdGenerator.GenerateNextChildIndex()
			}
		}
	}
	span.SetTag(trace.SPAN_ID, spanId)

	if lType == v2.EGRESS {
		span.SetTag(trace.APP_NAME, string(request.Header.Peek(models.APP_NAME)))
	}
	span.SetTag(trace.SPAN_TYPE, string(lType.(v2.ListenerType)))
	span.SetTag(trace.METHOD_NAME, string(request.Header.Peek(models.TARGET_METHOD)))
	span.SetTag(trace.PROTOCOL, "HTTP")
	span.SetTag(trace.SERVICE_NAME, string(request.Header.Peek(models.SERVICE_KEY)))
	span.SetTag(trace.BAGGAGE_DATA, string(request.Header.Peek(models.SOFA_TRACE_BAGGAGE_DATA)))

	return span
}
