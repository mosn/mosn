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

package rpc

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type engine struct {
	encoder types.Encoder
	decoder types.Decoder
}

type mixedEngine struct {
	engineMap map[byte]*engine
}

func NewEngine(encoder types.Encoder, decoder types.Decoder) types.ProtocolEngine {
	return &engine{
		encoder: encoder,
		decoder: decoder,
	}
}

func NewMixedEngine() types.ProtocolEngine {
	return &mixedEngine{
		engineMap: make(map[byte]*engine),
	}
}

func (eg *engine) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	return eg.encoder.Encode(ctx, model)
}

//func (eg *engine) EncodeTo(ctx context.Context, model interface{}, buf types.IoBuffer) (int, error) {
//	return eg.encoder.EncodeTo(ctx, model, buf)
//}

func (eg *engine) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	return eg.decoder.Decode(ctx, data)
}

func (eg *engine) Register(protocolCode byte, encoder types.Encoder, decoder types.Decoder) error {
	// unsupported for single protocol engine
	return nil
}

func (eg *engine) Process(ctx context.Context, data types.IoBuffer, handleFunc func(ctx2 context.Context, model interface{}, err error)) {

	for {
		cmd, err := eg.decoder.Decode(ctx, data)

		// No enough data
		if cmd == nil && err == nil {
			break
		}

		// Do handle staff. Error would also be passed to this function.
		handleFunc(ctx, cmd, err)
	}
}

func (m *mixedEngine) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	switch cmd := model.(type) {
	case RpcCmd:
		code := cmd.ProtocolCode()

		if eg, exists := m.engineMap[code]; exists {
			return eg.Encode(ctx, model)
		} else {
			return nil, ErrUnrecognizedCode
		}
	default:
		log.ByContext(ctx).Errorf("not RpcCmd, cannot find encoder for model = %+v", model)
		return nil, ErrUnknownType
	}
}

//func (m *mixedEngine) EncodeTo(ctx context.Context, model interface{}, buf types.IoBuffer) (int, error) {
//	switch cmd := model.(type) {
//	case RpcCmd:
//		code := cmd.ProtocolCode()
//
//		if eg, exists := m.engineMap[code]; exists {
//			return eg.EncodeTo(ctx, model, buf)
//		} else {
//			return 0, ErrUnrecognizedCode
//		}
//	default:
//
//		log.ByContext(ctx).Errorf("not RpcCmd, cannot find encoder for model = %+v", model)
//		return 0, ErrUnknownType
//	}
//}

func (m *mixedEngine) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	// at least 1 byte for protocol code recognize
	if data.Len() > 1 {
		logger := log.ByContext(ctx)
		code := data.Bytes()[0]
		logger.Debugf("mixed protocol engine decode, protocol code = %x", code)

		if eg, exists := m.engineMap[code]; exists {
			return eg.Decode(ctx, data)
		} else {
			return nil, ErrUnrecognizedCode
		}
	}
	return nil, nil
}

func (m *mixedEngine) Register(protocolCode byte, encoder types.Encoder, decoder types.Decoder) error {
	// register engine
	if _, exists := m.engineMap[protocolCode]; exists {
		return ErrDupRegistered
	} else {
		m.engineMap[protocolCode] = &engine{
			encoder: encoder,
			decoder: decoder,
		}
	}
	return nil
}

func (m *mixedEngine) Process(ctx context.Context, data types.IoBuffer, handleFunc func(ctx2 context.Context, model interface{}, err error)) {
	// at least 1 byte for protocol code recognize
	for data.Len() > 1 {
		logger := log.ByContext(ctx)
		code := data.Bytes()[0]
		logger.Debugf("mixed protocol engine process, protocol code = %x", code)

		if eg, exists := m.engineMap[code]; exists {
			cmd, err := eg.Decode(ctx, data)
			// No enough data
			if cmd == nil && err == nil {
				break
			}

			// Do handle staff. Error would also be passed to this function.
			handleFunc(ctx, cmd, err)
			if err != nil {
				return
			}
		} else {
			handleFunc(ctx, nil, ErrUnrecognizedCode)
			return
		}
	}
}
