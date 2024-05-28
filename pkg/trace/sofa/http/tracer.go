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

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/trace/sofa"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

type HTTPTracer struct {
	tracer api.Tracer
}

func NewTracer(config map[string]interface{}) (api.Tracer, error) {
	tracer, err := sofa.NewTracer(config)
	if err != nil {
		return nil, err
	}
	return &HTTPTracer{
		tracer: tracer,
	}, nil

}

func (t *HTTPTracer) Start(ctx context.Context, request interface{}, startTime time.Time) api.Span {
	span := t.tracer.Start(ctx, request, startTime)

	header, ok := request.(http.RequestHeader)
	if !ok || header.RequestHeader == nil {
		return span
	}

	t.HTTPDelegate(ctx, header, span)
	return span
}

func (t *HTTPTracer) HTTPDelegate(ctx context.Context, header http.RequestHeader, span api.Span) {
	traceId, ok := header.Get(sofa.HTTP_TRACER_ID_KEY)
	if !ok {
		traceId = trace.IdGen().GenerateTraceId()
	}
	span.SetTag(sofa.TRACE_ID, traceId)
	lType, _ := variable.Get(ctx, types.VariableListenerType)

	spanId, ok := header.Get(sofa.HTTP_RPC_ID_KEY)
	if !ok {
		spanId = "0" // Generate a new span id
	} else {
		if lType == v2.INGRESS {
			trace.AddSpanIdGenerator(trace.NewSpanIdGenerator(traceId, spanId))
		} else if lType == v2.EGRESS {
			span.SetTag(sofa.PARENT_SPAN_ID, spanId)
			spanKey := &trace.SpanKey{TraceId: traceId, SpanId: spanId}
			if spanIdGenerator := trace.GetSpanIdGenerator(spanKey); spanIdGenerator != nil {
				spanId = spanIdGenerator.GenerateNextChildIndex()
			}
		}
	}

	span.SetTag(sofa.SPAN_ID, spanId)

	if lType == v2.EGRESS {
		span.SetTag(sofa.CALLER_APP_NAME, string(header.Peek(sofa.APP_NAME_KEY)))
	}
	span.SetTag(sofa.SPAN_TYPE, string(lType.(v2.ListenerType)))
	span.SetTag(sofa.METHOD_NAME, string(header.Peek(sofa.TARGET_METHOD_KEY)))
	span.SetTag(sofa.PROTOCOL, "HTTP")
	span.SetTag(sofa.SERVICE_NAME, string(header.Peek(sofa.SERVICE_KEY)))
	span.SetTag(sofa.BAGGAGE_DATA, string(header.Peek(sofa.SOFA_TRACE_BAGGAGE_DATA)))

}
