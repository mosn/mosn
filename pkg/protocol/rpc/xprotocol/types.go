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

	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
)

// CodecFactory subprotocol plugin factory
type CodecFactory interface {
	CreateSubProtocolCodec(context context.Context) Multiplexing
}

type XCmd interface {
	rpc.RpcCmd

	//Tracing
	GetServiceName(data []byte) string
	GetMethodName(data []byte) string

	//RequestRouting
	GetMetas(data []byte) map[string]string

	//ProtocolConvertor
	Convert(data []byte) (map[string]string, []byte)
}

// SubProtocol Name
type SubProtocol string

// Multiplexing Accesslog Rate limit Curcuit Breakers
type Multiplexing interface {
	SplitFrame(data []byte) [][]byte
	GetStreamID(data []byte) string
	SetStreamID(data []byte, streamID string) []byte
}

// Tracing base on Multiplexing
type Tracing interface {
	Multiplexing
	GetServiceName(data []byte) string
	GetMethodName(data []byte) string
}

// RequestRouting RequestAccessControl RequesstFaultInjection base on Multiplexing
type RequestRouting interface {
	Multiplexing
	GetMetas(data []byte) map[string]string
}

// ProtocolConvertor change protocol base on Multiplexing
type ProtocolConvertor interface {
	Multiplexing
	Convert(data []byte) (map[string]string, []byte)
}
