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

package zipkin

import (
	"strconv"
	"time"

	"mosn.io/api"
	"mosn.io/pkg/log"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
)

type zipkinSpan struct {
	ztracer *zipkin.Tracer
	zspan   zipkin.Span
}

func (z zipkinSpan) TraceId() string {
	return z.zspan.Context().TraceID.String()
}

func (z zipkinSpan) SpanId() string {
	return z.zspan.Context().ID.String()
}

func (z zipkinSpan) ParentSpanId() string {
	return z.zspan.Context().ParentID.String()
}

func (z zipkinSpan) SetOperation(operation string) {
	z.zspan.SetName(operation)
}

func (z zipkinSpan) SetTag(key uint64, value string) {
	spanTag := SpanTag(int(key))
	z.zspan.Tag(spanTag.String(), value)
}

func (z zipkinSpan) SetRequestInfo(requestInfo api.RequestInfo) {
	z.zspan.Tag(RequestSize.String(), strconv.FormatUint(requestInfo.BytesReceived(), 10))
	z.zspan.Tag(ResponseSize.String(), strconv.FormatUint(requestInfo.BytesSent(), 10))

	if requestInfo.UpstreamHost() != nil {
		z.zspan.Tag(UpstreamHostAddress.String(), requestInfo.UpstreamHost().AddressString())
	}
	if requestInfo.DownstreamRemoteAddress() != nil {
		z.zspan.Tag(DownsteamHostAddress.String(), requestInfo.DownstreamRemoteAddress().String())
	}
	z.zspan.Tag(ResultStatus.String(), strconv.Itoa(requestInfo.ResponseCode()))
}

func (z zipkinSpan) Tag(key uint64) string {
	log.DefaultLogger.Debugf("[Zipkin] [tracer] [span] Unsupported Tag ")
	return ""
}

func (z zipkinSpan) FinishSpan() {
	z.zspan.Finish()
}

func (z zipkinSpan) InjectContext(request api.HeaderMap, requestInfo api.RequestInfo) {
	log.DefaultLogger.Debugf("[Zipkin] [tracer] [span] Unsupported InjectContext")
}

func (z zipkinSpan) SpawnChild(operationName string, startTime time.Time) api.Span {
	span := z.ztracer.StartSpan(operationName,
		zipkin.Parent(z.zspan.Context()),
		zipkin.Kind(model.Client),
		zipkin.StartTime(startTime),
	)
	return zipkinSpan{
		ztracer: z.ztracer,
		zspan:   span,
	}
}
