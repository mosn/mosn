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

package zipkin

import (
	"fmt"

	"mosn.io/api"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
)

const (
	DriverName = "Zipkin"
)

func init() {
	trace.RegisterDriver(DriverName, NewZipkinDriverImpl())
}

type holder struct {
	api.Tracer
	api.TracerBuilder
}

type zipkinDriver struct {
	tracers map[types.ProtocolName]*holder
}

func (z *zipkinDriver) Init(config map[string]interface{}) error {
	for proto, holder := range z.tracers {
		tracer, err := holder.TracerBuilder(config)
		if err != nil {
			return fmt.Errorf("build tracer for %v error, %s", proto, err)
		}
		holder.Tracer = tracer
	}
	return nil
}

func (z *zipkinDriver) Register(proto api.ProtocolName, builder api.TracerBuilder) {
	z.tracers[proto] = &holder{
		TracerBuilder: builder,
	}
}

func (z *zipkinDriver) Get(proto api.ProtocolName) api.Tracer {
	if holder, ok := z.tracers[proto]; ok {
		return holder.Tracer
	}
	return nil
}

func NewZipkinDriverImpl() *zipkinDriver {
	return &zipkinDriver{tracers: make(map[types.ProtocolName]*holder)}
}
