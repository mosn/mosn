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

package context

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/types"
)

const wrapperKey = "x-mosn-context"

func Get(ctx context.Context, key types.ContextKey) interface{} {
	if wrapperValue := ctx.Value(wrapperKey); wrapperValue != nil {
		if mosnCtx, ok := wrapperValue.(*Context); ok {
			return mosnCtx.builtin[key]
		}
	}
	return nil
}

func Set(ctx context.Context, key types.ContextKey, value interface{}) context.Context {
	if wrapperValue := ctx.Value(wrapperKey); wrapperValue != nil {
		if mosnCtx, ok := wrapperValue.(*Context); ok {
			mosnCtx.builtin[key] = value
		}
	} else {
		mosnCtx := &Context{}
		mosnCtx.builtin[key] = value
		ctx = context.WithValue(ctx, wrapperKey, mosnCtx)
	}

	return ctx
}

// Clone copy the origin mosn value context into new one
func Clone(ctx context.Context) context.Context {
	if wrapperValue := ctx.Value(wrapperKey); wrapperValue != nil {
		if mosnCtx, ok := wrapperValue.(*Context); ok {
			clone := *mosnCtx
			return context.WithValue(ctx, wrapperKey, &clone)
		}
	}
	return ctx
}
