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

package bolt

import (
	"context"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/trace/sofa"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

func Boltv1Delegate(ctx context.Context, frame api.XFrame, span api.Span) {
	request, ok := frame.(*bolt.Request)
	if !ok {
		log.Proxy.Errorf(ctx, "[protocol][sofarpc] boltv1 span build failed, type miss match:%+v", frame)
		return
	}

	header := request.GetHeader()

	traceId, ok := header.Get(sofa.TRACER_ID_KEY)
	if !ok {
		// TODO: set generated traceId into header?
		traceId = trace.IdGen().GenerateTraceId()
	}

	span.SetTag(sofa.TRACE_ID, traceId)
	lType, err := variable.Get(ctx, types.VariableListenerType)
	if err != nil || lType == nil {
		return
	}
	spanId, ok := header.Get(sofa.RPC_ID_KEY)
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

	appName, _ := header.Get(sofa.APP_NAME_KEY)
	span.SetTag(sofa.CALLER_APP_NAME, appName)
	span.SetTag(sofa.SPAN_TYPE, string(lType.(v2.ListenerType)))
	method, _ := header.Get(sofa.TARGET_METHOD_KEY)
	span.SetTag(sofa.METHOD_NAME, method)
	span.SetTag(sofa.PROTOCOL, string(bolt.ProtocolName))
	service, _ := header.Get(sofa.SERVICE_KEY)
	span.SetTag(sofa.SERVICE_NAME, service)
	bdata, _ := header.Get(sofa.SOFA_TRACE_BAGGAGE_DATA)
	span.SetTag(sofa.BAGGAGE_DATA, bdata)
	caller, _ := header.Get(sofa.CALLER_ZONE_KEY)
	span.SetTag(sofa.CALLER_CELL, caller)

}
