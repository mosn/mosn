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
	"log"
	"testing"
	"time"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/trace/sofa/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func TestSofaHttpTracerStartFinish(t *testing.T) {
	tracer, error := NewTracer(nil)
	if error != nil {
		log.Fatalln("create http tracer failed:", error)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, types.ContextKeyListenerType, v2.EGRESS)

	span := tracer.Start(ctx, nil, time.Now())
	span.SetTag(xprotocol.TRACE_ID, trace.IdGen().GenerateTraceId())
	span.FinishSpan()
}
