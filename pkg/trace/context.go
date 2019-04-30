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

package trace

import (
	"context"

	mosnctx "github.com/alipay/sofa-mosn/pkg/context"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type contextKey struct{}

type traceHolder struct {
	enableTracing bool
	tracer        types.Tracer
}

var holder = traceHolder{}

func SpanFromContext(ctx context.Context) types.Span {
	if val := mosnctx.Get(ctx, types.ContextKeyActiveSpan); val != nil {
		if sp, ok := val.(types.Span); ok {
			return sp
		}
	}

	return nil
}

func SetTracer(tracer types.Tracer) {
	holder.tracer = tracer
}

func Tracer() types.Tracer {
	return holder.tracer
}

func EnableTracing() {
	holder.enableTracing = true
}

func DisableTracing() {
	holder.enableTracing = false
}

func IsTracingEnabled() bool {
	return holder.enableTracing
}
