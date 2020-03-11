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

package ext

import (
	"context"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/rpc/sofarpc"
	"mosn.io/mosn/pkg/protocol/sofarpc/models"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/trace/sofa/rpc"
	"mosn.io/mosn/pkg/types"

	mosnctx "mosn.io/mosn/pkg/context"
)

func init() {
	rpc.RegisterSubProtocol(sofarpc.PROTOCOL_CODE_V1, boltv1Delegate)
}

func boltv1Delegate(ctx context.Context, cmd sofarpc.SofaRpcCmd, span types.Span) {
	request, ok := cmd.(*sofarpc.BoltRequest)
	if !ok {
		log.Proxy.Errorf(ctx, "[protocol][sofarpc] boltv1 span build failed, type missmatch:%+v", cmd)
		return
	}

	traceId := request.RequestHeader[models.TRACER_ID_KEY]
	if traceId == "" {
		// TODO: set generated traceId into header?
		traceId = trace.IdGen().GenerateTraceId()
	}

	span.SetTag(rpc.TRACE_ID, traceId)
	lType := mosnctx.Get(ctx, types.ContextKeyListenerType)
	if lType == nil {
		return
	}

	spanId := request.RequestHeader[models.RPC_ID_KEY]
	if spanId == "" {
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
	span.SetTag(rpc.APP_NAME, request.RequestHeader[models.APP_NAME])
	span.SetTag(rpc.SPAN_TYPE, string(lType.(v2.ListenerType)))
	span.SetTag(rpc.METHOD_NAME, request.RequestHeader[models.TARGET_METHOD])
	span.SetTag(rpc.PROTOCOL, "bolt")
	span.SetTag(rpc.SERVICE_NAME, request.RequestHeader[models.SERVICE_KEY])
	span.SetTag(rpc.BAGGAGE_DATA, request.RequestHeader[models.SOFA_TRACE_BAGGAGE_DATA])
	span.SetTag(rpc.CALLER_CELL, request.RequestHeader[models.CALLER_ZONE_KEY])
}
