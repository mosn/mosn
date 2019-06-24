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
	"time"

	"sofastack.io/sofa-mosn/pkg/types"
	"sofastack.io/sofa-mosn/pkg/trace"
	"sofastack.io/sofa-mosn/pkg/protocol"
	"context"
	"sofastack.io/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"sofastack.io/sofa-mosn/pkg/trace/sofa"
)

func init() {
	trace.RegisterTracerBuilder("SOFATracer", protocol.SofaRPC, NewTracer)
}

var PrintLog = true

type Tracer struct{}

func NewTracer(config map[string]interface{}) (types.Tracer, error) {
	// TODO: support log & report
	if PrintLog {
		// TODO: read logger path from config
		err := sofa.Init("", "", "")
		if err != nil {
			return nil, err
		}
	}

	return  &Tracer{}, nil
}

func (tracer *Tracer) Start(ctx context.Context, request interface{}, startTime time.Time) types.Span {
	span := NewSpan(startTime)

	cmd, ok := request.(sofarpc.SofaRpcCmd)
	if !ok || cmd == nil {
		return span
	}

	// ignore heartbeat
	if cmd.CommandCode() == sofarpc.HEARTBEAT {
		return span
	}

	// use delegate instrument if exists
	if delegate := delegateMap[cmd.ProtocolCode()]; delegate != nil {
		delegate(ctx, cmd, span)
	}
	return span
}
