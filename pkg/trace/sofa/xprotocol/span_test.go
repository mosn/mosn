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

package xprotocol

import (
	"log"
	"runtime"
	"testing"
)

func TestSpanLog(t *testing.T) {
	_, error := NewTracer(nil)
	if error != nil {
		log.Fatalln("create test tracer failed:", error)
	}

	span := &SofaRPCSpan{}
	span.log()
}

func TestIngressSpanLog(t *testing.T) {
	_, error := NewTracer(nil)
	if error != nil {
		log.Fatalln("create test tracer failed:", error)
	}

	span := &SofaRPCSpan{}
	span.tags[DOWNSTEAM_HOST_ADDRESS] = "127.0.0.1:43"
	span.tags[SPAN_TYPE] = "ingress"
	span.log()
}

func TestEgressSpanLog(t *testing.T) {
	_, error := NewTracer(nil)
	if error != nil {
		log.Fatalln("create test tracer failed:", error)
	}

	span := &SofaRPCSpan{}
	span.tags[SPAN_TYPE] = "egress"
	span.log()
}

func BenchmarkSofaTracelog(b *testing.B) {
	_, error := NewTracer(nil)
	if error != nil {
		log.Fatalln("create test tracer failed:", error)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	for n := 0; n < b.N; n++ {
		span := &SofaRPCSpan{}
		span.SetTag(TRACE_ID, "BenchmarkSofaTracer")
		span.SetTag(PARENT_SPAN_ID, "BenchmarkSofaTracer")
		span.SetTag(SERVICE_NAME, "BenchmarkSofaTracer")
		span.SetTag(METHOD_NAME, "BenchmarkSofaTracer")
		span.SetTag(PROTOCOL, "BenchmarkSofaTracer")
		span.SetTag(RESULT_STATUS, "BenchmarkSofaTracer")
		span.SetTag(REQUEST_SIZE, "BenchmarkSofaTracer")
		span.SetTag(RESPONSE_SIZE, "BenchmarkSofaTracer")
		span.SetTag(UPSTREAM_HOST_ADDRESS, "BenchmarkSofaTracer")
		span.SetTag(DOWNSTEAM_HOST_ADDRESS, "BenchmarkSofaTracer")
		span.SetTag(APP_NAME, "BenchmarkSofaTracer")
		span.SetTag(SPAN_TYPE, "BenchmarkSofaTracer")
		span.SetTag(BAGGAGE_DATA, "BenchmarkSofaTracer")
		span.SetTag(REQUEST_URL, "BenchmarkSofaTracer")
		span.SetTag(SPAN_TYPE, "ingress")

		span.FinishSpan()
	}
}

func BenchmarkSofaTracelogParallel(b *testing.B) {
	_, error := NewTracer(nil)
	if error != nil {
		log.Fatalln("create test tracer failed:", error)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			span := &SofaRPCSpan{}
			span.SetTag(TRACE_ID, "BenchmarkSofaTracer")
			span.SetTag(PARENT_SPAN_ID, "BenchmarkSofaTracer")
			span.SetTag(SERVICE_NAME, "BenchmarkSofaTracer")
			span.SetTag(METHOD_NAME, "BenchmarkSofaTracer")
			span.SetTag(PROTOCOL, "BenchmarkSofaTracer")
			span.SetTag(RESULT_STATUS, "BenchmarkSofaTracer")
			span.SetTag(REQUEST_SIZE, "BenchmarkSofaTracer")
			span.SetTag(RESPONSE_SIZE, "BenchmarkSofaTracer")
			span.SetTag(UPSTREAM_HOST_ADDRESS, "BenchmarkSofaTracer")
			span.SetTag(DOWNSTEAM_HOST_ADDRESS, "BenchmarkSofaTracer")
			span.SetTag(APP_NAME, "BenchmarkSofaTracer")
			span.SetTag(SPAN_TYPE, "BenchmarkSofaTracer")
			span.SetTag(BAGGAGE_DATA, "BenchmarkSofaTracer")
			span.SetTag(REQUEST_URL, "BenchmarkSofaTracer")
			span.SetTag(SPAN_TYPE, "ingress")

			span.FinishSpan()
		}
	})
}
