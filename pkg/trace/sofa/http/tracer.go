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

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"

	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/trace/sofa"
	"mosn.io/mosn/pkg/trace/sofa/xprotocol"
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
	span := xprotocol.NewSpan(startTime)

	header, ok := request.(http.RequestHeader)
	if !ok || header.RequestHeader == nil {
		return span
	}

	traceId, ok := header.Get(sofa.HTTP_TRACER_ID_KEY)
	if !ok {
		traceId = trace.IdGen().GenerateTraceId()
	}
	span.SetTag(xprotocol.TRACE_ID, traceId)
	lType := mosnctx.Get(ctx, types.ContextKeyListenerType)

	spanId, ok := header.Get(sofa.HTTP_RPC_ID_KEY)
	if !ok {
		spanId = "0" // Generate a new span id
	} else {
		if lType == v2.INGRESS {
			trace.AddSpanIdGenerator(trace.NewSpanIdGenerator(traceId, spanId))
		} else if lType == v2.EGRESS {
			span.SetTag(xprotocol.PARENT_SPAN_ID, spanId)
			spanKey := &trace.SpanKey{TraceId: traceId, SpanId: spanId}
			if spanIdGenerator := trace.GetSpanIdGenerator(spanKey); spanIdGenerator != nil {
				spanId = spanIdGenerator.GenerateNextChildIndex()
			}
		}
	}
	span.SetTag(xprotocol.SPAN_ID, spanId)

	if lType == v2.EGRESS {
		span.SetTag(xprotocol.APP_NAME, string(header.Peek(sofa.APP_NAME)))
	}
	span.SetTag(xprotocol.SPAN_TYPE, string(lType.(v2.ListenerType)))
	span.SetTag(xprotocol.METHOD_NAME, string(header.Peek(sofa.TARGET_METHOD)))
	span.SetTag(xprotocol.PROTOCOL, "HTTP")
	span.SetTag(xprotocol.SERVICE_NAME, string(header.Peek(sofa.SERVICE_KEY)))
	span.SetTag(xprotocol.BAGGAGE_DATA, string(header.Peek(sofa.SOFA_TRACE_BAGGAGE_DATA)))

	return span
}
