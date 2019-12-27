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

	"mosn.io/mosn/pkg/types"
)

func Get(ctx context.Context, key types.ContextKey) interface{} {
	if mosnCtx, ok := ctx.(*valueCtx); ok {
		return mosnCtx.builtin[key]
	}
	return ctx.Value(key)
}

// WithValue add the given key-value pair into the existed value context, or create a new value context which contains the pair.
// This Function should not be used along with the official context.WithValue !!

// The following context topology will leads to existed pair {'foo':'bar'} NOT FOUND, because recursive lookup for
// key-type=ContextKey is not supported by mosn.valueCtx.
//
// topology: context.Background -> mosn.valueCtx{'foo':'bar'} -> context.valueCtx -> mosn.valueCtx{'hmm':'haa'}
func WithValue(parent context.Context, key types.ContextKey, value interface{}) context.Context {
	if mosnCtx, ok := parent.(*valueCtx); ok {
		mosnCtx.builtin[key] = value
		return mosnCtx
	}

	// create new valueCtx
	mosnCtx := &valueCtx{Context: parent}
	mosnCtx.builtin[key] = value
	return mosnCtx
}

// Clone copy the origin mosn value context(if it is), and return new one
func Clone(parent context.Context) context.Context {
	if mosnCtx, ok := parent.(*valueCtx); ok {
		clone := &valueCtx{Context: mosnCtx}
		// array copy assign
		clone.builtin = mosnCtx.builtin
		return clone
	}
	return parent
}
