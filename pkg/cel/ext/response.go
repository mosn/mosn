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

package ext

import (
	"context"

	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
)

func addResponseheader(ctx context.Context, key, val string) bool {
	headers, ok := mosnctx.Get(ctx, types.ContextKeyDownStreamRespHeaders).(api.HeaderMap)
	if !ok {
		return false
	}

	// TODO support append mode
	headers.Set(key, val)
	return true
}

func delResponseheader(ctx context.Context, key string) bool {
	headers, ok := mosnctx.Get(ctx, types.ContextKeyDownStreamRespHeaders).(api.HeaderMap)
	if !ok {
		return false
	}

	headers.Del(key)
	return true
}
