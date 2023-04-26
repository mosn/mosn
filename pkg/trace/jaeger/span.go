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

package jaeger

import (
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

type Span struct {
	jaegerSpan opentracing.Span
	spanCtx    jaeger.SpanContext
}

func (s *Span) TraceId() string {
	return s.spanCtx.TraceID().String()
}

func (s *Span) SpanId() string {
	return s.spanCtx.SpanID().String()
}

func (s *Span) ParentSpanId() string {
	return s.spanCtx.ParentID().String()
}

func (s *Span) SetOperation(operation string) {
	s.jaegerSpan.SetOperationName(operation)
}

func (s *Span) SetTag(key uint64, value string) {
	log.DefaultLogger.Debugf("[Jaeger] [tracer] [span] Unsupported SetTag [%d]-[%s]", key, value)
}

func (s *Span) Tag(key uint64) string {
	log.DefaultLogger.Debugf("[Jaeger] [tracer] [span] Unsupported Tag [%d]-[%s]", key)
	return ""
}

// SetRequestInfo record current request info to Span
func (s *Span) SetRequestInfo(reqinfo api.RequestInfo) {
	span := s.jaegerSpan
	span.SetTag("request_size", reqinfo.BytesReceived())
	span.SetTag("response_size", reqinfo.BytesSent())
	if reqinfo.UpstreamHost() != nil {
		span.SetTag("upstream_host_address", reqinfo.UpstreamHost().AddressString())
	}
	if reqinfo.DownstreamLocalAddress() != nil {
		span.SetTag("downstream_host_address", reqinfo.DownstreamRemoteAddress().String())
	}

	code := reqinfo.ResponseCode()
	span.SetTag("http.status_code", code)

	if isErrorResponse(code) {
		span.SetTag("error", true)
		span.SetTag("error.message", http.StatusText(code))
	}
	span.SetTag("mosn_process_time", reqinfo.ProcessTimeDuration().String())
}

func isErrorResponse(code int) bool {
	if code >= http.StatusInternalServerError && code <= http.StatusNetworkAuthenticationRequired {
		return true
	}
	return false
}

// FinishSpan submit span info to agent
func (s *Span) FinishSpan() {
	s.jaegerSpan.Finish()
}

func (s *Span) InjectContext(requestHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	requestHeaders.Set(jaeger.TraceContextHeaderName, s.spanCtx.String())

	service := ""
	if value, ok := requestHeaders.Get(HeaderRouteMatchKey); ok {
		service = value
	}

	s.jaegerSpan.SetTag(HeaderRouteMatchKey, service)
}

func (s *Span) SpawnChild(operationName string, startTime time.Time) api.Span {
	log.DefaultLogger.Debugf("[Jaeger] [tracer] [span] Unsupported SpawnChild [%s]", operationName)
	return nil
}
