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

package http

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/internal/tool"
	"github.com/SkyAPM/go2sky/propagation"
	v3 "github.com/SkyAPM/go2sky/reporter/grpc/language-agent"
)

const (
	errInvalidTracer = tool.Error("invalid tracer")
)

const componentIDGOHttpServer = 5004

type handler struct {
	tracer    *go2sky.Tracer
	name      string
	next      http.Handler
	extraTags map[string]string
}

// ServerOption allows Middleware to be optionally configured.
type ServerOption func(*handler)

// Tag adds extra tag to server spans.
func WithServerTag(key string, value string) ServerOption {
	return func(h *handler) {
		if h.extraTags == nil {
			h.extraTags = make(map[string]string)
		}
		h.extraTags[key] = value
	}
}

// WithOperationName override default operation name.
func WithServerOperationName(name string) ServerOption {
	return func(h *handler) {
		h.name = name
	}
}

// NewServerMiddleware returns a http.Handler middleware with tracing.
func NewServerMiddleware(tracer *go2sky.Tracer, options ...ServerOption) (func(http.Handler) http.Handler, error) {
	if tracer == nil {
		return nil, errInvalidTracer
	}
	return func(next http.Handler) http.Handler {
		h := &handler{
			tracer: tracer,
			next:   next,
		}
		for _, option := range options {
			option(h)
		}
		return h
	}, nil
}

// ServeHTTP implements http.Handler.
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	span, ctx, err := h.tracer.CreateEntrySpan(r.Context(), getOperationName(h.name, r), func() (string, error) {
		return r.Header.Get(propagation.Header), nil
	})
	if err != nil {
		if h.next != nil {
			h.next.ServeHTTP(w, r)
		}
		return
	}
	span.SetComponent(componentIDGOHttpServer)
	for k, v := range h.extraTags {
		span.Tag(go2sky.Tag(k), v)
	}
	span.Tag(go2sky.TagHTTPMethod, r.Method)
	span.Tag(go2sky.TagURL, fmt.Sprintf("%s%s", r.Host, r.URL.Path))
	span.SetSpanLayer(v3.SpanLayer_Http)

	rww := &responseWriterWrapper{w: w, statusCode: 200}
	defer func() {
		code := rww.statusCode
		if code >= 400 {
			span.Error(time.Now(), "Error on handling request")
		}
		span.Tag(go2sky.TagStatusCode, strconv.Itoa(code))
		span.End()
	}()
	if h.next != nil {
		h.next.ServeHTTP(rww, r.WithContext(ctx))
	}
}

type responseWriterWrapper struct {
	w          http.ResponseWriter
	statusCode int
}

func (rww *responseWriterWrapper) Header() http.Header {
	return rww.w.Header()
}

func (rww *responseWriterWrapper) Write(bytes []byte) (int, error) {
	return rww.w.Write(bytes)
}

func (rww *responseWriterWrapper) WriteHeader(statusCode int) {
	rww.statusCode = statusCode
	rww.w.WriteHeader(statusCode)
}

func getOperationName(name string, r *http.Request) string {
	if name == "" {
		return fmt.Sprintf("/%s%s", r.Method, r.URL.Path)
	}
	return name
}
