// Licensed to SkyAPM org under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. SkyAPM org licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package go2sky

import (
	"context"

	"github.com/SkyAPM/go2sky/internal/idgen"
	"github.com/pkg/errors"

	"github.com/SkyAPM/go2sky/internal/tool"
	"github.com/SkyAPM/go2sky/propagation"
)

const (
	errParameter = tool.Error("parameter are nil")
	EmptyTraceID = "N/A"
	NoopTraceID  = "[Ignored Trace]"
)

// Tracer is go2sky tracer implementation.
type Tracer struct {
	service  string
	instance string
	reporter Reporter
	// 0 not init 1 init
	initFlag int32
}

// TracerOption allows for functional options to adjust behaviour
// of a Tracer to be created by NewTracer
type TracerOption func(t *Tracer)

// NewTracer return a new go2sky Tracer
func NewTracer(service string, opts ...TracerOption) (tracer *Tracer, err error) {
	if service == "" {
		return nil, errParameter
	}
	t := &Tracer{
		service:  service,
		initFlag: 0,
	}
	for _, opt := range opts {
		opt(t)
	}

	if t.reporter != nil {
		if t.instance == "" {
			id, err := idgen.UUID()
			if err != nil {
				return nil, err
			}
			t.instance = id + "@" + tool.IPV4()
		}
		t.reporter.Boot(t.service, t.instance)
		t.initFlag = 1
	}
	return t, nil
}

// CreateEntrySpan creates and starts an entry span for incoming request
func (t *Tracer) CreateEntrySpan(ctx context.Context, operationName string, extractor propagation.Extractor) (s Span, nCtx context.Context, err error) {
	if ctx == nil || operationName == "" || extractor == nil {
		return nil, nil, errParameter
	}
	if s, nCtx = t.createNoop(ctx); s != nil {
		return
	}
	header, err := extractor()
	if err != nil {
		return
	}
	var refSc *propagation.SpanContext
	if header != "" {
		refSc = &propagation.SpanContext{}
		err = refSc.DecodeSW8(header)
		if err != nil {
			return
		}
	}
	s, nCtx, err = t.CreateLocalSpan(ctx, WithContext(refSc), WithSpanType(SpanTypeEntry))
	if err != nil {
		return
	}
	s.SetOperationName(operationName)
	return
}

// CreateLocalSpan creates and starts a span for local usage
func (t *Tracer) CreateLocalSpan(ctx context.Context, opts ...SpanOption) (s Span, c context.Context, err error) {
	if ctx == nil {
		return nil, nil, errParameter
	}
	if s, c = t.createNoop(ctx); s != nil {
		return
	}
	ds := newLocalSpan(t)
	for _, opt := range opts {
		opt(ds)
	}
	parentSpan, ok := ctx.Value(ctxKeyInstance).(segmentSpan)
	if !ok {
		parentSpan = nil
	}
	s, err = newSegmentSpan(ds, parentSpan)
	if err != nil {
		return nil, nil, err
	}
	return s, context.WithValue(ctx, ctxKeyInstance, s), nil
}

// CreateExitSpan creates and starts an exit span for client
func (t *Tracer) CreateExitSpan(ctx context.Context, operationName string, peer string, injector propagation.Injector) (Span, error) {
	if ctx == nil || operationName == "" || peer == "" || injector == nil {
		return nil, errParameter
	}
	if s, _ := t.createNoop(ctx); s != nil {
		return s, nil
	}
	s, _, err := t.CreateLocalSpan(ctx, WithSpanType(SpanTypeExit))
	if err != nil {
		return nil, err
	}
	s.SetOperationName(operationName)
	s.SetPeer(peer)
	spanContext := &propagation.SpanContext{}
	span, ok := s.(ReportedSpan)
	if !ok {
		return nil, errors.New("span type is wrong")
	}

	firstSpan := span.Context().FirstSpan
	spanContext.Sample = 1
	spanContext.TraceID = span.Context().TraceID
	spanContext.ParentSegmentID = span.Context().SegmentID
	spanContext.ParentSpanID = span.Context().SpanID
	spanContext.ParentService = t.service
	spanContext.ParentServiceInstance = t.instance
	spanContext.ParentEndpoint = firstSpan.GetOperationName()
	spanContext.AddressUsedAtClient = peer

	err = injector(spanContext.EncodeSW8())
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (t *Tracer) createNoop(ctx context.Context) (s Span, nCtx context.Context) {
	if ns, ok := ctx.Value(ctxKeyInstance).(*NoopSpan); ok {
		nCtx = ctx
		s = ns
		return
	}
	if t.initFlag == 0 {
		s = &NoopSpan{}
		nCtx = context.WithValue(ctx, ctxKeyInstance, s)
		return
	}
	return
}

type ctxKey struct{}

var ctxKeyInstance = ctxKey{}

//Reporter is a data transit specification
type Reporter interface {
	Boot(service string, serviceInstance string)
	Send(spans []ReportedSpan)
	Close()
}

func TraceID(ctx context.Context) string {
	activeSpan := ctx.Value(ctxKeyInstance)
	if activeSpan == nil {
		return EmptyTraceID
	}
	span, ok := activeSpan.(segmentSpan)
	if ok {
		return span.context().TraceID
	}
	return NoopTraceID
}
