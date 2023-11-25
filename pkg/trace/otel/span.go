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

package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/http"
)

type Span struct {
	otelSpan trace.Span
	nctx     context.Context
	pctx     context.Context
}

func (s *Span) TraceId() string {
	return s.otelSpan.SpanContext().TraceID().String()
}

func (s *Span) SpanId() string {
	return s.otelSpan.SpanContext().SpanID().String()
}

func (s *Span) ParentSpanId() string {
	spanContext := trace.SpanContextFromContext(s.pctx)
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.SpanID().String()
}

// SetRequestInfo record current request info to Span
func (s *Span) SetRequestInfo(reqInfo api.RequestInfo) {
	s.otelSpan.SetAttributes(attribute.Int("http.response.status_code", reqInfo.ResponseCode()))

	if a := reqInfo.DownstreamLocalAddress(); a != nil {
		s.otelSpan.SetAttributes(attribute.String("downstream.local.address", a.String()))
	}

	if a := reqInfo.DownstreamRemoteAddress(); a != nil {
		s.otelSpan.SetAttributes(attribute.String("downstream.remote.address", a.String()))
	}
}

func (s *Span) SetRequestHeader(header http.RequestHeader) {
	s.otelSpan.SetAttributes(attribute.String("network.protocol.name", string(header.Protocol())))
	s.otelSpan.SetAttributes(attribute.Key("http.method").String(string(header.Method())))
	s.otelSpan.SetAttributes(attribute.Key("http.url").String(string(header.RequestURI())))
}

func (s *Span) FinishSpan() {
	s.otelSpan.End()
}

func (s *Span) InjectContext(requestHeaders api.HeaderMap, _ api.RequestInfo) {
	otel.GetTextMapPropagator().Inject(s.nctx, HTTPHeadersCarrier{requestHeaders})
}

/* unsupported method */

func (s *Span) SetOperation(operation string) {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[otel] [tracer] [span] Unsupported SetOperation [%s]", operation)
	}
}

func (s *Span) SetTag(key uint64, value string) {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[otel] [tracer] [span] Unsupported SetTag [%d]-[%s]", key, value)
	}
}

func (s *Span) Tag(key uint64) string {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[otel] [tracer] [span] Unsupported Tag [%d]-[%s]", key)
	}
	return ""
}

func (s *Span) SpawnChild(operationName string, _ time.Time) api.Span {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[otel] [tracer] [span] Unsupported SpawnChild [%s]", operationName)
	}
	return nil
}
