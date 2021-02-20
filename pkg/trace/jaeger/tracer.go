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
	serviceName        = "service_name"
	agentHost          = "agent_host"
	defaultServiceName = "default_sidecar"
	jaegerAgentHostKey = "TRACE"
	appIDKey           = "APP_ID"
)

func init() {
	trace.RegisterTracerBuilder(DriverName, protocol.HTTP1, NewTracer)
}

type Tracer struct {
	tracer opentracing.Tracer
}

//NewTracer create new jaeger
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

	tracer, _, err := cfg.New(getServiceName(traceCfg))

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

	//如果配置文件没有设置，则取环境变量TRACE
	if host := os.Getenv(jaegerAgentHostKey); host != "" {
		return host
	}

	return "0.0.0.0:6831"
}

func getServiceName(traceCfg map[string]interface{}) string {
	if service, ok := traceCfg[serviceName]; ok {
		return service.(string)
	}

	//如果配置文件没有设置，则取环境变量APP_ID
	if appID := os.Getenv(appIDKey); appID != "" {
		return fmt.Sprintf("%s_sidecar", appID)
	}

	return defaultServiceName
}

//Start 初始化tracer中的span
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

	sp, ctx := opentracing.StartSpanFromContextWithTracer(ctx, t.tracer, getOperationName(header.RequestURI()), opentracing.ChildOf(spanCtx), opentracing.StartTime(startTime))

	ctx = opentracing.ContextWithSpan(ctx, sp)

	return sp, spanCtx
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
