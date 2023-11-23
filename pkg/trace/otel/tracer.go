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
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"os"
	"time"

	"mosn.io/api"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	mosntrace "mosn.io/mosn/pkg/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	CONSOLE = "console"
	HTTP    = "http"
	GRPC    = "grpc"
)

const (
	tracerName         = "mosn.io/mosn/pkg/trace/otel"
	defaultServiceName = "default_sidecar"
)

func init() {
	mosntrace.RegisterTracerBuilder(DriverName, protocol.HTTP1, NewTracer)
}

type Tracer struct {
	tracer trace.Tracer
}

// Start init span
func (t *Tracer) Start(ctx context.Context, request interface{}, startTime time.Time) api.Span {
	header, ok := request.(http.RequestHeader)
	if !ok || header.RequestHeader == nil {
		log.DefaultLogger.Errorf("[otel] [tracer] [http1] unable to get request header, downstream trace ignored")
		return &Span{}
	}

	sp, nctx, pctx := t.getSpan(ctx, header, startTime)
	log.DefaultLogger.Infof("[otel] [tracer] [http1] traceId: %s, spanId: %s.", sp.SpanContext().TraceID().String(), sp.SpanContext().SpanID().String())

	span := Span{otelSpan: sp, nctx: nctx, pctx: pctx}
	span.SetRequestHeader(header)

	return &span
}

func (t *Tracer) getSpan(ctx context.Context, header http.RequestHeader, startTime time.Time) (trace.Span, context.Context, context.Context) {
	// 包含上游 trace 信息的 pctx（与原 ctx 不是一个）
	pctx := otel.GetTextMapPropagator().Extract(ctx, HTTPHeadersCarrier{header})

	options := make([]trace.SpanStartOption, 0)
	options = append(options, trace.WithTimestamp(startTime))
	options = append(options, trace.WithSpanKind(trace.SpanKindServer))
	if !trace.SpanContextFromContext(pctx).IsValid() {
		// 不包含上游 trace
		options = append(options, trace.WithNewRoot())
	}

	// 包含当前 Span 的 ctx
	nctx, span := t.tracer.Start(pctx, generateSpanName(header), options...)

	return span, nctx, pctx
}

//goland:noinspection GoImportUsedAsName
func generateSpanName(header http.RequestHeader) string {
	protocol := string(header.Protocol())
	method := string(header.Method())
	url := string(header.RequestURI())

	// Http/1.1 Get /
	return fmt.Sprintf("%s %s %s", protocol, method, url)
}

func NewTracer(traceCfg map[string]interface{}) (api.Tracer, error) {
	config, err := NewTracerConfig(traceCfg)
	if err != nil {
		return nil, err
	}

	r, err := config.GetResource()
	if err != nil {
		return nil, err
	}

	exp, err := config.GetExporter()
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(r))

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(getPropagator())

	return &Tracer{tracer: tp.Tracer(tracerName, trace.WithSchemaURL(semconv.SchemaURL), trace.WithInstrumentationVersion("1.0.0"))}, nil
}

func getPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	)
}

type TracerConfig struct {
	ServiceName  string            `json:"service_name"`
	ReportMethod string            `json:"report_method"`
	Endpoint     string            `json:"endpoint"`
	ConfigMap    map[string]string `json:"config_map"`
}

func (c *TracerConfig) GetServiceName() string {
	if c.ServiceName != "" {
		return c.ServiceName
	}

	//if service_name is not set, get it from the env variable
	if appID := os.Getenv("COMP_NAME"); appID != "" {
		c.ServiceName = fmt.Sprintf("%s_sidecar", appID)
	} else {
		c.ServiceName = defaultServiceName
	}

	return c.ServiceName
}

func (c *TracerConfig) GetResource() (*resource.Resource, error) {
	attributes := make([]attribute.KeyValue, 0)
	attributes = append(attributes, attribute.String("service.name", c.GetServiceName()))

	if c.ConfigMap != nil {
		for key, value := range c.ConfigMap {
			attributes = append(attributes, attribute.Key(key).String(value))
		}
	}

	return resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		attributes...,
	))
}

func (c *TracerConfig) GetExporter() (sdktrace.SpanExporter, error) {

	if c.ReportMethod == "" {
		c.ReportMethod = CONSOLE
	}

	if c.ReportMethod != CONSOLE && c.Endpoint == "" {
		c.ReportMethod = CONSOLE
	}

	switch c.ReportMethod {
	case HTTP:
		return otlptracehttp.New(context.Background(), otlptracehttp.WithEndpoint(c.Endpoint), otlptracehttp.WithInsecure())
	case GRPC:
		return otlptracegrpc.New(context.Background(), otlptracegrpc.WithEndpoint(c.Endpoint), otlptracegrpc.WithInsecure())
	case CONSOLE:
		fallthrough
	default:
		return stdouttrace.New()
	}
}

func NewTracerConfig(traceCfg map[string]interface{}) (*TracerConfig, error) {
	cfgJson, err := json.Marshal(traceCfg)
	if err != nil {
		return nil, err
	}

	cfgObj := new(TracerConfig)
	err = json.Unmarshal(cfgJson, cfgObj)

	return cfgObj, err
}
