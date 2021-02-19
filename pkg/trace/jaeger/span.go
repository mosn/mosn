package jaeger

import (
	"fmt"
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
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
	s.jaegerSpan.SetTag(string(key), value)
}

//SetRequestInfo 把当前请求相关信息放入span中
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

// Tag 返回当前span所指定的tag
func (s *Span) Tag(key uint64) string {
	return ""
}

//FinishSpan 完成span，提交信息到agent
func (s *Span) FinishSpan() {
	s.jaegerSpan.Finish()
}

func (s *Span) InjectContext(requestHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	span := s.jaegerSpan
	//把当前创建的span的trace string设置到header头中，parent_span_id = current span id
	traceString := fmt.Sprintf("%s", s.jaegerSpan)
	requestHeaders.Set(jaeger.TraceContextHeaderName, traceString)

	service := ""
	if value, ok := requestHeaders.Get(HeaderRouteMatchKey); ok {
		service = value
	}

	span.SetTag(HeaderRouteMatchKey, service)
}

func (s *Span) SpawnChild(operationName string, startTime time.Time) types.Span {
	return nil
}
