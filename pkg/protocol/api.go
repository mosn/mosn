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

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol/internal/registry"
	"mosn.io/mosn/pkg/types"
)

var (
	protocolsSupported = map[api.ProtocolName]struct{}{
		Auto: struct{}{}, // reserved protocol, support for Auto protocol config parsed
	}
)

// A MOSN Protocol contains three parts:
// 1. A Factory to create connection pool defines by types.ConnectionPool
// 2. An interface to match protocol and create protocol stream for each request.
// 3. A status code mapping handler to translate protocol status code to HTTP status code, which
// is used as standard status code in MOSN.
func RegisterProtocol(name api.ProtocolName, newPool types.NewConnPool, streamFactory types.ProtocolStreamFactory, mapping api.HTTPMapping) error {
	if _, ok := protocolsSupported[name]; ok {
		return ErrDuplicateProtocol
	}
	registry.RegisterNewPoolFactory(name, newPool)
	registry.ResgiterProtocolStreamFactory(name, streamFactory)
	registry.RegisterMapping(name, mapping)
	protocolsSupported[name] = struct{}{}
	return nil
}

// The api for get the registered info
var (
	ErrNoMapping         = errors.New("no mapping function found")
	ErrDuplicateProtocol = errors.New("duplicated protocol registered")
)

func ProtocolRegistered(name api.ProtocolName) bool {
	_, ok := protocolsSupported[name]
	return ok
}

func RangeAllRegisteredProtocol(f func(name api.ProtocolName)) {
	for proto := range protocolsSupported {
		if proto != Auto {
			f(proto)
		}
	}
}

func MappingHeaderStatusCode(ctx context.Context, p api.ProtocolName, headers api.HeaderMap) (int, error) {
	if f, ok := registry.HttpMappingFactory[p]; ok {
		return f.MappingHeaderStatusCode(ctx, headers)
	}
	return 0, ErrNoMapping
}

func GetNewPoolFactory(protocol api.ProtocolName) (types.NewConnPool, bool) {
	factory, ok := registry.ConnNewPoolFactories[protocol]
	return factory, ok
}

func GetProtocolStreamFactory(name api.ProtocolName) (types.ProtocolStreamFactory, bool) {
	f, ok := registry.StreamFactories[name]
	return f, ok
}

// GetProtocolStreamFactories returns a slice of ProtocolStreamFactory by the name.
// if the names is empty, return all the registered ProtocolStreamFactory
func GetProtocolStreamFactories(names []api.ProtocolName) map[api.ProtocolName]types.ProtocolStreamFactory {
	if len(names) == 0 {
		// do a copy to avoid be modified
		m := make(map[api.ProtocolName]types.ProtocolStreamFactory, len(registry.StreamFactories))
		for p, factory := range registry.StreamFactories {
			m[p] = factory
		}
		return m
	}

	m := make(map[api.ProtocolName]types.ProtocolStreamFactory, len(names))
	for _, p := range names {
		factory, ok := registry.StreamFactories[p]
		if ok {
			m[p] = factory
		}
	}
	return m
}
