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

package rpc

import (
	"time"

	"context"

	"sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/protocol"
	"sofastack.io/sofa-mosn/pkg/protocol/http"
	"sofastack.io/sofa-mosn/pkg/protocol/sofarpc/models"
	"sofastack.io/sofa-mosn/pkg/trace"
	"sofastack.io/sofa-mosn/pkg/trace/sofa/rpc"
	"sofastack.io/sofa-mosn/pkg/types"

	mosnctx "sofastack.io/sofa-mosn/common/context"
)

func init() {
	trace.RegisterTracerBuilder("SOFATracer", protocol.HTTP1, NewTracer)
}

var PrintLog = true

type Tracer struct{}

func NewTracer(config map[string]interface{}) (types.Tracer, error) {
	// inherit rpc's logger & output format
	return &Tracer{}, nil
}

func (tracer *Tracer) Start(ctx context.Context, request interface{}, startTime time.Time) types.Span {
	span := rpc.NewSpan(startTime)

	header, ok := request.(http.RequestHeader)
	if !ok || header.RequestHeader == nil {
		return span
	}

	traceId, ok := header.Get(models.HTTP_TRACER_ID_KEY)
	if !ok {
		traceId = trace.IdGen().GenerateTraceId()
	}
	span.SetTag(rpc.TRACE_ID, traceId)
	lType := mosnctx.Get(ctx, mosnctx.ContextKeyListenerType)

	spanId, ok := header.Get(models.HTTP_RPC_ID_KEY)
	if !ok {
		spanId = "0" // Generate a new span id
	} else {
		if lType == v2.INGRESS {
			trace.AddSpanIdGenerator(trace.NewSpanIdGenerator(traceId, spanId))
		} else if lType == v2.EGRESS {
			span.SetTag(rpc.PARENT_SPAN_ID, spanId)
			spanKey := &trace.SpanKey{TraceId: traceId, SpanId: spanId}
			if spanIdGenerator := trace.GetSpanIdGenerator(spanKey); spanIdGenerator != nil {
				spanId = spanIdGenerator.GenerateNextChildIndex()
			}
		}
	}
	span.SetTag(rpc.SPAN_ID, spanId)

	if lType == v2.EGRESS {
		span.SetTag(rpc.APP_NAME, string(header.Peek(models.APP_NAME)))
	}
	span.SetTag(rpc.SPAN_TYPE, string(lType.(v2.ListenerType)))
	span.SetTag(rpc.METHOD_NAME, string(header.Peek(models.TARGET_METHOD)))
	span.SetTag(rpc.PROTOCOL, "HTTP")
	span.SetTag(rpc.SERVICE_NAME, string(header.Peek(models.SERVICE_KEY)))
	span.SetTag(rpc.BAGGAGE_DATA, string(header.Peek(models.SOFA_TRACE_BAGGAGE_DATA)))

	return span
}
