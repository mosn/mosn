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
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"mosn.io/api"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/trace"
)

const (
	serviceName            = "service_name"
	agentHost              = "agent_host"
	defaultServiceName     = "default_sidecar"
	defaultJaegerAgentHost = "0.0.0.0:6831"
	jaegerAgentHostKey     = "TRACE"
	appIDKey               = "APP_ID"
)

func init() {
	trace.RegisterTracerBuilder(DriverName, protocol.HTTP1, NewTracer)
}

type Tracer struct {
	tracer opentracing.Tracer
}

// NewTracer create new jaeger
func NewTracer(traceCfg map[string]interface{}) (api.Tracer, error) {
	cfg := config.Configuration{
		Disabled: false,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            false,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  getAgentHost(traceCfg),
		},
	}

	cfg.ServiceName = getServiceName(traceCfg)
	tracer, _, err := cfg.NewTracer()

	log.DefaultLogger.Infof("[jaeger] [tracer] jaeger agent host:%s, report service name:%s",
		getAgentHost(traceCfg), getServiceName(traceCfg))

	if err != nil {
		log.DefaultLogger.Errorf("[jaeger] [tracer] [http1] cannot initialize Jaeger Tracer")
		return nil, err
	}

	return &Tracer{
		tracer: tracer,
	}, nil
}

func getAgentHost(traceCfg map[string]interface{}) string {
	if agentHost, ok := traceCfg[agentHost]; ok {
		return agentHost.(string)
	}

	//if TRACE is not set, get it from the env variable
	if host := os.Getenv(jaegerAgentHostKey); host != "" {
		return host
	}

	return defaultJaegerAgentHost
}

func getServiceName(traceCfg map[string]interface{}) string {
	if service, ok := traceCfg[serviceName]; ok {
		return service.(string)
	}

	//if service_name is not set, get it from the env variable
	if appID := os.Getenv(appIDKey); appID != "" {
		return fmt.Sprintf("%s_sidecar", appID)
	}

	return defaultServiceName
}

// Start init span
func (t *Tracer) Start(ctx context.Context, request interface{}, startTime time.Time) api.Span {
	header, ok := request.(http.RequestHeader)

	if !ok || header.RequestHeader == nil {
		log.DefaultLogger.Errorf("[Jaeger] [tracer] [http1] unable to get request header, downstream trace ignored")
		return &Span{}
	}

	sp, spanCtx := t.getSpan(ctx, header, startTime)

	ext.HTTPMethod.Set(sp, string(header.Method()))
	ext.HTTPUrl.Set(sp, string(header.RequestURI()))

	return &Span{
		jaegerSpan: sp,
		spanCtx:    spanCtx,
	}
}

func (t *Tracer) getSpan(ctx context.Context, header http.RequestHeader, startTime time.Time) (opentracing.Span, jaeger.SpanContext) {
	httpHeaderPropagator := jaeger.NewHTTPHeaderPropagator(getDefaultHeadersConfig(), *jaeger.NewNullMetrics())

	spanCtx, _ := httpHeaderPropagator.Extract(HTTPHeadersCarrier(header))

	sp, _ := opentracing.StartSpanFromContextWithTracer(ctx, t.tracer, getOperationName(header.RequestURI()), opentracing.ChildOf(spanCtx), opentracing.StartTime(startTime))

	//renew span context
	newSpanCtx, ok := sp.Context().(jaeger.SpanContext)
	if !ok {
		return sp, spanCtx
	}

	return sp, newSpanCtx
}

// split url, example:http://127.0.0.1:8080?query=11
func getOperationName(uri []byte) string {
	arr := strings.Split(string(uri), "?")
	return arr[0]
}

func getDefaultHeadersConfig() *jaeger.HeadersConfig {
	return &jaeger.HeadersConfig{
		JaegerDebugHeader:        jaeger.JaegerDebugHeader,
		JaegerBaggageHeader:      jaeger.JaegerBaggageHeader,
		TraceContextHeaderName:   jaeger.TraceContextHeaderName,
		TraceBaggageHeaderPrefix: jaeger.TraceBaggageHeaderPrefix,
	}
}
