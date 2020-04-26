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

package protocol

import (
	"context"
	"errors"
	"strconv"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

var (
	httpMappingFactory = make(map[api.Protocol]HTTPMapping)
	ErrNoMapping       = errors.New("no mapping function found")
)

// HTTP and HTTP2 does not need mapping
func init() {
	RegisterMapping(HTTP1, &httpMapping{})
	RegisterMapping(HTTP2, &httpMapping{})
}

// HTTPMapping maps the contents of protocols to HTTP standard
type HTTPMapping interface {
	MappingHeaderStatusCode(ctx context.Context, headers api.HeaderMap) (int, error)
}

func RegisterMapping(p api.Protocol, m HTTPMapping) {
	httpMappingFactory[p] = m
}

func MappingHeaderStatusCode(ctx context.Context, p api.Protocol, headers api.HeaderMap) (int, error) {
	if f, ok := httpMappingFactory[p]; ok {
		return f.MappingHeaderStatusCode(ctx, headers)
	}
	return 0, ErrNoMapping
}

// HTTP get status directly
type httpMapping struct{}

func (m *httpMapping) MappingHeaderStatusCode(ctx context.Context, headers api.HeaderMap) (int, error) {
	status, ok := headers.Get(types.HeaderStatus)
	if !ok {
		return 0, errors.New("headers have no status code")
	}
	return strconv.Atoi(status)
}
