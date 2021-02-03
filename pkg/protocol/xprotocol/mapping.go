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
	"context"
	"errors"

	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

var (
	ErrNoSubProtocolFound = errors.New("no sub protocol found in context")
	ErrNoDelegateFound    = errors.New("no sub protocol delegate function found")
)

func init() {
	protocol.RegisterMapping(protocol.Xprotocol, &xprotocolMapping{})
}

type xprotocolMapping struct{}

func (m *xprotocolMapping) MappingHeaderStatusCode(ctx context.Context, headers types.HeaderMap) (int, error) {
	// 1. get sub-protocol from context
	subProtocol := mosnctx.Get(ctx, types.ContextSubProtocol)
	if subProtocol == nil {
		return 0, ErrNoSubProtocolFound
	}

	// 2. get delegate function for the sub-protocol
	delegate := GetMapping(types.ProtocolName(subProtocol.(string)))
	if delegate == nil {
		return 0, ErrNoDelegateFound
	}

	// 3. invoke
	return delegate.MappingHeaderStatusCode(ctx, headers)
}
