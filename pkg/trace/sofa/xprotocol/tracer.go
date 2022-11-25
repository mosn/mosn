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
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/trace/sofa"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

type ProtocolDelegate func(ctx context.Context, frame api.XFrame, span api.Span)

var (
	delegates = make(map[api.ProtocolName]ProtocolDelegate)
)

func RegisterDelegate(name api.ProtocolName, delegate ProtocolDelegate) {
	delegates[name] = delegate
}

func GetDelegate(name api.ProtocolName) ProtocolDelegate {
	return delegates[name]
}

type XTracer struct {
	tracer api.Tracer
}

func NewTracer(config map[string]interface{}) (api.Tracer, error) {
	tracer, err := sofa.NewTracer(config)
	if err != nil {
		return nil, err
	}
	return &XTracer{
		tracer: tracer,
	}, nil
}

func (t *XTracer) Start(ctx context.Context, frame interface{}, startTime time.Time) api.Span {
	span := t.tracer.Start(ctx, frame, startTime)

	xframe, ok := frame.(api.XFrame)
	if !ok || xframe == nil {
		return span
	}

	// ignore heartbeat
	if xframe.IsHeartbeatFrame() {
		return span
	}

	// the trace protocol is based on request (downstream)
	var proto api.ProtocolName
	v, err := variable.Get(ctx, types.VariableDownStreamProtocol)
	if err != nil {
		return span
	}
	if proto, ok = v.(api.ProtocolName); !ok {
		return span
	}

	if delegate := GetDelegate(proto); delegate != nil {
		delegate(ctx, xframe, span)
	}

	return span
}
