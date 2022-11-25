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
	"mosn.io/pkg/variable"
)

// ProtocolName type definition
const (
	Auto  api.ProtocolName = "Auto"
	HTTP1 api.ProtocolName = "Http1" // TODO: move it to protocol/HTTP
	HTTP2 api.ProtocolName = "Http2" // TODO: move it to protocol/HTTP2

)

// header direction definition
const (
	Request  = "Request"
	Response = "Response"
)

func init() {
	variable.Register(variable.NewStringVariable(types.VarDirection, nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable(types.VarScheme, nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable(types.VarHost, nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable(types.VarPath, nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable(types.VarPathOriginal, nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable(types.VarQueryString, nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable(types.VarMethod, nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable(types.VarIstioHeaderHost, nil, nil, variable.DefaultStringSetter, 0))
}

// TODO: use pkg.CommonHeader, why not?
// CommonHeader wrapper for map[string]string
type CommonHeader map[string]string

// Get value of key
func (h CommonHeader) Get(key string) (value string, ok bool) {
	value, ok = h[key]
	return
}

// Set key-value pair in header map, the previous pair will be replaced if exists
func (h CommonHeader) Set(key string, value string) {
	h[key] = value
}

// Add value for given key.
// Multiple headers with the same key may be added with this function.
// Use Set for setting a single header for the given key.
func (h CommonHeader) Add(key string, value string) {
	panic("not supported")
}

// Del delete pair of specified key
func (h CommonHeader) Del(key string) {
	delete(h, key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (h CommonHeader) Range(f func(key, value string) bool) {
	for k, v := range h {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
}

// Clone used to deep copy header's map
func (h CommonHeader) Clone() api.HeaderMap {
	copy := make(map[string]string)

	for k, v := range h {
		copy[k] = v
	}

	return CommonHeader(copy)
}

func (h CommonHeader) ByteSize() uint64 {
	var size uint64

	for k, v := range h {
		size += uint64(len(k) + len(v))
	}
	return size
}

var ErrNoStatusCode = errors.New("headers have no status code")

// GetStatusCodeMapping is a default http mapping implementation, which get
// status code from variable without any translation.
type GetStatusCodeMapping struct{}

var _ api.HTTPMapping = (*GetStatusCodeMapping)(nil)

func (m GetStatusCodeMapping) MappingHeaderStatusCode(ctx context.Context, headers api.HeaderMap) (int, error) {
	status, err := variable.GetString(ctx, types.VarHeaderStatus)
	if err != nil {
		return 0, ErrNoStatusCode
	}
	return strconv.Atoi(status)
}
