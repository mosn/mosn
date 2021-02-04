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

package skywalking

import (
	"context"
	"testing"
	"time"

	"github.com/SkyAPM/go2sky"
	"mosn.io/api"

	"mosn.io/mosn/pkg/types"
)

type mockTracer struct {
	*go2sky.Tracer
}

func (m mockTracer) Start(ctx context.Context, request interface{}, startTime time.Time) api.Span {
	return nil
}

func (m *mockTracer) SetGO2SkyTracer(tracer *go2sky.Tracer) {
	m.Tracer = tracer
}

func TestTraceBuilderRegisterAndGet(t *testing.T) {
	driver := NewSkyDriverImpl()
	proto := types.ProtocolName("test")

	driver.Register(proto, func(config map[string]interface{}) (tracer api.Tracer, err error) {
		return &mockTracer{}, nil
	})

	// use default config
	err := driver.Init(nil)
	if err != nil {
		t.Error(err.Error())
	}

	tracer := driver.Get(proto)
	if tracer == nil {
		t.Error("get tracer from driver failed")
	}
	if mockT, ok := tracer.(mockTracer); ok {
		if mockT.Tracer == nil {
			t.Errorf("no injection go2sky.Tracer")
		}
	}
}
