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

package skywalking

import (
	"time"

	"github.com/SkyAPM/go2sky"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

const (
	noopTraceID  = "[Ignored Trace]"
	spanID       = "0"
	parentSpanID = "-1"
)

var NoopSpan = noopSkySpan{}

type SkySpan struct {
}

func (SkySpan) SpanId() string {
	return spanID
}

func (SkySpan) ParentSpanId() string {
	return parentSpanID
}

func (SkySpan) SetOperation(operation string) {
	log.DefaultLogger.Debugf("[SkyWalking] [tracer] [span] Unsupported SetOperation [%s]", operation)
}

func (SkySpan) SetTag(key uint64, value string) {
	log.DefaultLogger.Debugf("[SkyWalking] [tracer] [span] Unsupported SetTag [%d]-[%s]", key, value)
}

func (SkySpan) Tag(key uint64) string {
	log.DefaultLogger.Debugf("[SkyWalking] [tracer] [span] Unsupported Tag [%s]", key)
	return ""
}

func (SkySpan) SpawnChild(operationName string, _ time.Time) api.Span {
	log.DefaultLogger.Debugf("[SkyWalking] [tracer] [span] Unsupported SpawnChild [%s]", operationName)
	return nil
}

type noopSkySpan struct {
	SkySpan
}

func (s noopSkySpan) TraceId() string {
	return noopTraceID
}

func (s noopSkySpan) SetRequestInfo(_ api.RequestInfo) {
}

func (s noopSkySpan) FinishSpan() {
}

func (s noopSkySpan) InjectContext(_ types.HeaderMap, _ types.RequestInfo) {
}

// SpanCarrier save the entry span and exit span in one request
type SpanCarrier struct {
	EntrySpan go2sky.Span
	ExitSpan  go2sky.Span
}
