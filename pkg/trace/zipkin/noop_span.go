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

package zipkin

import (
	"time"

	"mosn.io/api"
)

const (
	NoopSpanTraceID  = "-1"
	NoopSpanID       = "0"
	NoopSpanParentID = "-1"
)

var NoopSpan = noopSpan{}

type noopSpan struct{}

func (n noopSpan) TraceId() string {
	return NoopSpanTraceID
}

func (n noopSpan) SpanId() string {
	return NoopSpanID
}

func (n noopSpan) ParentSpanId() string {
	return NoopSpanParentID
}

func (n noopSpan) SetOperation(operation string) {
}

func (n noopSpan) SetTag(key uint64, value string) {
}

func (n noopSpan) SetRequestInfo(requestInfo api.RequestInfo) {
}

func (n noopSpan) Tag(key uint64) string {
	return ""
}

func (n noopSpan) FinishSpan() {
}

func (n noopSpan) InjectContext(requestHeaders api.HeaderMap, requestInfo api.RequestInfo) {
}

func (n noopSpan) SpawnChild(operationName string, startTime time.Time) api.Span {
	return n
}
