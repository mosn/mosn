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

package trace

import (
	"context"
	"testing"
	"time"

	"mosn.io/api"

	"mosn.io/mosn/pkg/types"
)

type mockTracer struct {
}

func (tracer *mockTracer) Start(ctx context.Context, request interface{}, startTime time.Time) api.Span {
	return nil
}

func TestTraceBuilderRegisterAndGet(t *testing.T) {
	driver := NewDefaultDriverImpl()
	proto := types.ProtocolName("test")

	driver.Register(proto, func(config map[string]interface{}) (api.Tracer, error) {
		return &mockTracer{}, nil
	})

	driver.Init(nil)

	tracer := driver.Get(proto)

	if tracer == nil {
		t.Error("get tracer from driver failed")
	}

}
