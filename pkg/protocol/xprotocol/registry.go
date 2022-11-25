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

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/internal/registry"
	"mosn.io/mosn/pkg/types"
)

var (
	xNewConnpool      func(ctx context.Context, codec api.XProtocolCodec, host types.Host) types.ConnectionPool
	xNewStreamFactory func(codec api.XProtocolCodec) types.ProtocolStreamFactory
	xExtends          func(codec api.XProtocolCodec)
)

// RegisterXProtocolAction register the xprotocol actions that used to
// register into protocol.
// The actions contain:
// 1. A function to create a connection pool
// 2. A function to create a stream factory
// 3. A function to extends, for example, tracer register
func RegisterXProtocolAction(newConnpool func(ctx context.Context, codec api.XProtocolCodec, host types.Host) types.ConnectionPool,
	newStreamFactory func(codec api.XProtocolCodec) types.ProtocolStreamFactory,
	extends func(codec api.XProtocolCodec)) {
	xNewConnpool = newConnpool
	xNewStreamFactory = newStreamFactory
	xExtends = extends
}

// RegisterXProtocolCodec register the xprotocol to protocol.
func RegisterXProtocolCodec(codec api.XProtocolCodec) error {

	if xNewConnpool == nil || xNewStreamFactory == nil {
		return errors.New("xprotocol actions is not registered, xprotocol register failed")
	}

	name := codec.ProtocolName()

	if err := registry.RegisterXProtocolCodec(name, codec); err != nil {
		return err
	}

	// register the protocol
	if err := protocol.RegisterProtocol(name, func(ctx context.Context, host types.Host) types.ConnectionPool {
		return xNewConnpool(ctx, codec, host)
	}, xNewStreamFactory(codec), codec.HTTPMapping()); err != nil {
		return err
	}

	if xExtends != nil {
		xExtends(codec)
	}

	return nil
}
