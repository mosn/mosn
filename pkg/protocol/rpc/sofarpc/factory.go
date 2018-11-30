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

package sofarpc

import (
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var (
	sofarpcEngine = rpc.NewMixedEngine()

	// Consider the use case of heartbeat construction: an individual scene which can't get the reference
	// of the protocol engine instance(usually it held by the stream connection), like upstream/healthcheck.
	// So we need a separate factory to achieve this goal.
	heartbeatFactory = make(map[byte]HeartbeatBuilder)

	responseFactory = make(map[byte]ResponseBuilder)
)

func RegisterProtocol(protocolCode byte, encoder types.Encoder, decoder types.Decoder, spanBuilder types.SpanBuilder) {
	sofarpcEngine.Register(protocolCode, encoder, decoder, spanBuilder)
}

func RegisterHeartbeatBuilder(protocolCode byte, builder HeartbeatBuilder) {
	heartbeatFactory[protocolCode] = builder
}

func RegisterResponseBuilder(protocolCode byte, builder ResponseBuilder) {
	responseFactory[protocolCode] = builder
}

// TODO: should be replaced with configure specify(e.g. downstream_protocol: rpc, sub_protocol:[boltv1])
func Engine() types.ProtocolEngine {
	return sofarpcEngine
}
