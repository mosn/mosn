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
	"time"

	"context"

	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/trace/sofa"
	"mosn.io/mosn/pkg/types"
)

func init() {
	trace.RegisterTracerBuilder("SOFATracer", protocol.Xprotocol, NewTracer)
}

var PrintLog = true

type Tracer struct{}

func NewTracer(config map[string]interface{}) (types.Tracer, error) {
	// TODO: support log & report
	if PrintLog {
		if value, ok := config["log_path"]; ok {
			if logPath, ok := value.(string); ok {
				if err := sofa.Init(protocol.Xprotocol, logPath, "rpc-server-digest.log", "rpc-client-digest.log"); err != nil {
					return nil, err
				}
			}
		} else {
			err := sofa.Init(protocol.Xprotocol, "", "rpc-server-digest.log", "rpc-client-digest.log")
			if err != nil {
				return nil, err
			}
		}
	}

	return &Tracer{}, nil
}

func (tracer *Tracer) Start(ctx context.Context, frame interface{}, startTime time.Time) types.Span {
	span := NewSpan(startTime)

	xframe, ok := frame.(xprotocol.XFrame)
	if !ok || xframe == nil {
		return span
	}

	// ignore heartbeat
	if xframe.IsHeartbeatFrame() {
		return span
	}

	// use delegate instrument if exists
	subProtocol := types.ProtocolName(mosnctx.Get(ctx, types.ContextSubProtocol).(string))

	if delegate := delegateMap[subProtocol]; delegate != nil {
		delegate(ctx, xframe, span)
	}
	return span
}
