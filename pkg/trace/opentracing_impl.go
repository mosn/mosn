package trace

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"strconv"
	"time"
)

var spanChan = make(chan *SimpleOpenTracingSpan)

func init() {
	go func() {
		for {
			span := <-spanChan
			fmt.Printf("Time:%s;%v\n", time.Now().Format(time.RFC3339), span)
		}
	}()
}

type SimpleOpenTracingSpan struct {
	startTime time.Time
	endTime   time.Time
	traceId   string
	spanId    string
	tags      map[string]interface{}
}

func (span *SimpleOpenTracingSpan) Finish() {
	span.endTime = time.Now()
	spanChan <- span
}

func (span *SimpleOpenTracingSpan) FinishWithOptions(opts opentracing.FinishOptions) {

}

func (span *SimpleOpenTracingSpan) Context() opentracing.SpanContext {
	return nil
}

func (span *SimpleOpenTracingSpan) SetOperationName(operationName string) opentracing.Span {
	return nil
}

func (span *SimpleOpenTracingSpan) SetTag(key string, value interface{}) opentracing.Span {
	if key == TRACE_ID {
		if value.(string) == "" {
			span.traceId = IdGen().GenerateTraceId()
		} else {
			span.traceId = value.(string)
		}
	} else if key == SPAN_ID {
		if value.(string) == "" {
			span.spanId = "0.1"
		} else {
			span.spanId = value.(string)
		}
	} else {
		span.tags[key] = value
	}

	return span
}

func (span *SimpleOpenTracingSpan) String() string {
	return fmt.Sprintf("TraceId:%s;SpanId:%s;Duration:%s;Protocol:%s;ServiceName:%s;requestSize:%s;responseSize:%s;upstreamHostAddress:%s;downstreamRemoteHostAdress:%s",
		span.traceId,
		span.spanId,
		strconv.FormatInt(span.endTime.Sub(span.startTime).Nanoseconds()/1000000, 10),
		span.tags[PROTOCOL],
		span.tags[SERVICE_NAME],
		span.tags[REQUEST_SIZE],
		span.tags[RESPONSE_SIZE],
		span.tags[UPSTREAM_HOST_ADDRESS],
		span.tags[DOWNSTEAM_HOST_ADDRESS])
}

func (span *SimpleOpenTracingSpan) LogFields(fields ...log.Field) {

}

func (span *SimpleOpenTracingSpan) LogKV(alternatingKeyValues ...interface{}) {

}

func (span *SimpleOpenTracingSpan) SetBaggageItem(restrictedKey, value string) opentracing.Span {
	return nil
}

func (span *SimpleOpenTracingSpan) BaggageItem(restrictedKey string) string {
	return ""
}

func (span *SimpleOpenTracingSpan) Tracer() opentracing.Tracer {
	return nil
}

func (span *SimpleOpenTracingSpan) LogEvent(event string) {

}

func (span *SimpleOpenTracingSpan) LogEventWithPayload(event string, payload interface{}) {

}

func (span *SimpleOpenTracingSpan) Log(data opentracing.LogData) {

}
