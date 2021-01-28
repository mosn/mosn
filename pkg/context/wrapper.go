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
	pkgcontext "mosn.io/pkg/context"
)

// Deprecated: use mosn.io/pkg/context/wrapper.go:Get instead
var Get = pkgcontext.Get

// WithValue add the given key-value pair into the existed value context, or create a new value context which contains the pair.
// This Function should not be used along with the official context.WithValue !!

// The following context topology will leads to existed pair {'foo':'bar'} NOT FOUND, because recursive lookup for
// key-type=ContextKey is not supported by mosn.valueCtx.
//
// topology: context.Background -> mosn.valueCtx{'foo':'bar'} -> context.valueCtx -> mosn.valueCtx{'hmm':'haa'}
// Deprecated: use mosn.io/pkg/context/wrapper.go:WithValue instead
var WithValue = pkgcontext.WithValue

// Clone copy the origin mosn value context(if it is), and return new one
// Deprecated: use mosn.io/pkg/context/wrapper.go:Clone instead
var Clone = pkgcontext.Clone
