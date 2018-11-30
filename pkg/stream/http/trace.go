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

package http

import (
	"time"

	"github.com/alipay/sofa-mosn/pkg/trace"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/valyala/fasthttp"
)

var spanBuilder = &SpanBuilder{}

type SpanBuilder struct {
}

func (spanBuilder *SpanBuilder) BuildSpan(args ...interface{}) types.Span {
	if len(args) == 0 {
		return nil
	}

	if requestCtx, ok := args[0].(fasthttp.RequestCtx); ok {
		span := trace.Tracer().Start(time.Now())
		span.SetTag(trace.PROTOCOL, "http")
		span.SetTag(trace.METHOD_NAME, string(requestCtx.Method()))
		span.SetTag(trace.REQUEST_URL, string(requestCtx.Host())+string(requestCtx.Path()))
		span.SetTag(trace.REQUEST_SIZE, "0") // TODO
		return span
	}

	return nil
}
