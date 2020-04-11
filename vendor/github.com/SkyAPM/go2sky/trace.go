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
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/SkyAPM/go2sky/internal/tool"
	"github.com/SkyAPM/go2sky/propagation"
)

const (
	errParameter = tool.Error("parameter are nil")
	EmptyTraceID = "N/A"
	NoopTraceID  = "[Ignored Trace]"
	// -1 represent the object doesn't exist.
	Inexistence = -1
)

// Tracer is go2sky tracer implementation.
type Tracer struct {
	service  string
	instance string
	reporter Reporter
	// 0 not init 1 init
	initFlag   int32
	serviceID  int32
	instanceID int32
	wg         *sync.WaitGroup
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
		service:    service,
		initFlag:   0,
		serviceID:  0,
		instanceID: 0,
	}
	for _, opt := range opts {
		opt(t)
	}
	if t.reporter != nil {
		if t.instance == "" {
			id, err := uuid.NewUUID()
			if err != nil {
				return nil, err
			}
			t.instance = id.String()
		}
		t.wg = &sync.WaitGroup{}
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			for {
				serviceID, instanceID, err := t.reporter.Register(t.service, t.instance)
				if err != nil {
					time.Sleep(5 * time.Second)
					continue
				}
				if atomic.SwapInt32(&t.serviceID, serviceID) == 0 && atomic.SwapInt32(&t.instanceID, instanceID) == 0 {
					atomic.SwapInt32(&t.initFlag, 1)
					break
				}
			}
		}()
	}
	return t, nil
}

//WaitUntilRegister is a tool helps user to wait until register process has finished
func (t *Tracer) WaitUntilRegister() {
	if t.wg != nil {
		t.wg.Wait()
	}
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
		err = refSc.DecodeSW6(header)
		if err != nil {
			return
		}
	}
	s, nCtx, err = t.CreateLocalSpan(ctx, WithContext(refSc), WithSpanType(SpanTypeEntry))
	if err != nil {
		return
	}
	s.SetOperationName(operationName)
	ref, ok := nCtx.Value(refKeyInstance).(*propagation.SpanContext)
	if ok && ref != nil {
		return
	}
	sc := &propagation.SpanContext{
		Sample:                 1,
		ParentEndpoint:         operationName,
		EntryEndpoint:          operationName,
		EntryServiceInstanceID: t.instanceID,
	}
	if refSc != nil {
		sc.Sample = refSc.Sample
		if refSc.EntryEndpoint != "" {
			sc.EntryEndpoint = refSc.EntryEndpoint
		}
		sc.EntryEndpointID = refSc.EntryEndpointID
		sc.EntryServiceInstanceID = refSc.EntryServiceInstanceID
	}
	nCtx = context.WithValue(nCtx, refKeyInstance, sc)
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
	s = newSegmentSpan(ds, parentSpan)
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
	spanContext.Sample = 1
	spanContext.TraceID = span.Context().TraceID
	spanContext.ParentSpanID = span.Context().SpanID
	spanContext.ParentSegmentID = span.Context().SegmentID
	spanContext.NetworkAddress = peer
	spanContext.ParentServiceInstanceID = t.instanceID

	// Since 6.6.0, if first span is not entry span, then this is an internal segment(no RPC), rather than an endpoint.
	// EntryEndpoint
	firstSpan := span.Context().FirstSpan
	firstSpanOperationName := firstSpan.GetOperationName()
	ref, ok := ctx.Value(refKeyInstance).(*propagation.SpanContext)
	var entryEndpoint = ""
	var entryServiceInstanceID int32
	if ok && ref != nil {
		spanContext.Sample = ref.Sample
		entryEndpoint = ref.EntryEndpoint
		entryServiceInstanceID = ref.EntryServiceInstanceID
	} else {
		if firstSpan.IsEntry() {
			entryEndpoint = firstSpanOperationName
		} else {
			spanContext.EntryEndpointID = Inexistence
		}
		entryServiceInstanceID = t.instanceID
	}
	spanContext.EntryServiceInstanceID = entryServiceInstanceID
	if entryEndpoint != "" {
		spanContext.EntryEndpoint = entryEndpoint
	}
	// ParentEndpoint
	if firstSpan.IsEntry() && firstSpanOperationName != "" {
		spanContext.ParentEndpoint = firstSpanOperationName
	} else {
		spanContext.ParentEndpointID = Inexistence
	}

	err = injector(spanContext.EncodeSW6())
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

type refKey struct{}

var ctxKeyInstance = ctxKey{}

var refKeyInstance = refKey{}

//Reporter is a data transit specification
type Reporter interface {
	Register(service string, instance string) (int32, int32, error)
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
		return span.context().GetReadableGlobalTraceID()
	}
	return NoopTraceID
}
