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
	"testing"

	"github.com/stretchr/testify/require"
	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

type mockProtocolStreamFactory struct {
	types.ProtocolStreamFactory
}

type mockMapping struct{}

func (m *mockMapping) MappingHeaderStatusCode(ctx context.Context, headers api.HeaderMap) (int, error) {
	return 200, nil
}

func TestRegisterProtocol(t *testing.T) {
	names := []api.ProtocolName{
		api.ProtocolName("testprotocol"),
		api.ProtocolName("testprotocol2"),
	}
	for _, name := range names {
		RegisterProtocol(name, func(ctx context.Context, host types.Host) types.ConnectionPool {
			return nil
		}, &mockProtocolStreamFactory{}, &mockMapping{})
	}

	ps := 0
	RangeAllRegisteredProtocol(func(_ api.ProtocolName) {
		ps++
	})
	require.Equal(t, 2, ps)

	for _, name := range names {
		require.True(t, ProtocolRegistered(name))
		status, _ := MappingHeaderStatusCode(context.Background(), name, nil)
		require.Equal(t, 200, status)
		_, ok := GetNewPoolFactory(name)
		require.True(t, ok)
		_, ok = GetProtocolStreamFactory(name)
		require.True(t, ok)
	}

	unknown := api.ProtocolName("unknown")

	_, err := MappingHeaderStatusCode(context.Background(), unknown, nil)
	require.ErrorIs(t, err, ErrNoMapping)

	factories := GetProtocolStreamFactories(nil)
	require.Len(t, factories, 2)
	f := GetProtocolStreamFactories([]api.ProtocolName{unknown})
	require.Len(t, f, 0)
	f2 := GetProtocolStreamFactories([]api.ProtocolName{names[0]})
	require.Len(t, f2, 1)
}

func TestRegisterProtocolWithOutMapping(t *testing.T) {
	name := api.ProtocolName("testprotocol3")
	RegisterProtocol(name, func(ctx context.Context, host types.Host) types.ConnectionPool {
		return nil
	}, &mockProtocolStreamFactory{}, nil)

	require.True(t, ProtocolRegistered(name))

	_, err := MappingHeaderStatusCode(context.Background(), name, nil)
	require.ErrorIs(t, err, ErrNoMapping)
}
