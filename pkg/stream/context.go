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

package stream

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

// contextManager
type ContextManager struct {
	base context.Context
	curr context.Context
}

func (cm *ContextManager) Get() context.Context {
	return cm.curr
}

func (cm *ContextManager) Next() {
	// buffer context
	cm.curr = buffer.NewBufferPoolContext(cm.base)
	// variable context
	cm.curr = variable.NewVariableContext(cm.curr)
}

func (cm *ContextManager) InjectTrace(ctx context.Context, span api.Span) context.Context {
	if span != nil {
		_ = variable.Set(ctx, types.VariableTraceId, span.TraceId())
		return ctx
	}
	// generate traceId
	_ = variable.Set(ctx, types.VariableTraceId, trace.IdGen().GenerateTraceId())
	return ctx
}

func NewContextManager(base context.Context) *ContextManager {
	return &ContextManager{
		base: base,
	}
}
