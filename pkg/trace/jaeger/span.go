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

//SetRequestInfo record current request info to Span
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

//FinishSpan submit span info to agent
func (s *Span) FinishSpan() {
	s.jaegerSpan.Finish()
}

func (s *Span) InjectContext(requestHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	traceString := s.spanCtx.String()
	requestHeaders.Set(jaeger.TraceContextHeaderName, traceString)

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
