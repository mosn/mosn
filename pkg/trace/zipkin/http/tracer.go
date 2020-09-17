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

	zipkintracer "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/trace/zipkin"
	"mosn.io/mosn/pkg/types"
)

func init() {
	trace.RegisterTracerBuilder(zipkin.ZipkinTracer, protocol.HTTP1, NewHttpZipkinTracer)
}

func NewHttpZipkinTracer(conf map[string]interface{}) (types.Tracer, error) {
	return &Tracer{}, nil
}

type Tracer struct {
	tracer *zipkintracer.Tracer
}

func (t *Tracer) SetGO2ZipkinTracer(tracer *zipkintracer.Tracer) {
	t.tracer = tracer
}

func (t *Tracer) Start(ctx context.Context, request interface{}, startTime time.Time) types.Span {
	if t.tracer == nil {
		t.tracer, _ = zipkintracer.NewTracer(zipkinreporter.NewNoopReporter())
	}

	var carries zipkin.Carrier
	var spanCtx *model.SpanContext
	switch h := request.(type) {
	case zipkin.HeaderMap:
		carries = &zipkin.HeaderCarrier{HeaderMap: h}
	case zipkin.Carrier:
		carries = h
	}
	if carries != nil {
		ctx, err := zipkin.ExtractB3(carries)
		if err == nil {
			spanCtx = ctx
		}
	}

	return zipkin.NewSpan(spanCtx, t.tracer, carries, startTime)
}
