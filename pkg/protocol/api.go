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
// 1. A Factory to create connection pool defined by types.ConnectionPool
// 2. An interface to match protocol and create protocol stream for each request.
// 3. A status code mapping handler to translate protocol status code to HTTP status code, which
// is used as standard status code in MOSN.
func RegisterProtocol(name api.ProtocolName, newPool types.NewConnPool, streamFactory types.ProtocolStreamFactory, mapping api.HTTPMapping) error {
	if _, ok := protocolsSupported[name]; ok {
		return ErrDuplicateProtocol
	}
	if newPool == nil || streamFactory == nil {
		return ErrInvalidParameters
	}
	registry.RegisterNewPoolFactory(name, newPool)
	registry.RegisterProtocolStreamFactory(name, streamFactory)
	registry.RegisterMapping(name, mapping)
	protocolsSupported[name] = struct{}{}
	return nil
}

// The api for get the registered info
var (
	ErrNoMapping         = errors.New("no mapping function found")
	ErrInvalidParameters = errors.New("new connection pool or protocol stream factory cannot be nil")
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

var (
	FAILED = errors.New("FAILED")
	EAGAIN = errors.New("AGAIN")
)

// SelectStreamFactoryProtocol match the protocol.
// if scopes is nil, match all the registered protocols
// if scopes is not nil, match the protocol in the scopes
func SelectStreamFactoryProtocol(ctx context.Context, prot string, peek []byte, scopes []api.ProtocolName) (api.ProtocolName, error) {
	var err error
	var again bool
	if len(scopes) == 0 {
		// if scopes is nil, match all the registered protocols
		for p, factory := range registry.StreamFactories {
			err = factory.ProtocolMatch(ctx, prot, peek)
			if err == nil {
				return p, nil
			}
			if err == EAGAIN {
				again = true
			}
		}
	} else {
		// if scopes is not nil,  match the protocol in the scopes
		for _, p := range scopes {
			factory, ok := registry.StreamFactories[p]
			if !ok {
				continue
			}
			err = factory.ProtocolMatch(ctx, prot, peek)
			if err == nil {
				return p, nil
			}
			if err == EAGAIN {
				again = true
			}
		}
	}
	if again {
		return "", EAGAIN
	}
	return "", FAILED
}
