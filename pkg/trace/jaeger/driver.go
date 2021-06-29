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

package jaeger

import (
	"fmt"

	"mosn.io/api"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
)

const (
	DriverName          = "jaeger"
	HeaderRouteMatchKey = "service"
)

func init() {
	trace.RegisterDriver(DriverName, NewJaegerImpl())
}

type holder struct {
	api.Tracer
	api.TracerBuilder
}

type jaegerDriver struct {
	tracers map[types.ProtocolName]*holder
}

func (d *jaegerDriver) Init(traceCfg map[string]interface{}) error {
	for proto, holder := range d.tracers {
		tracer, err := holder.TracerBuilder(traceCfg)
		if err != nil {
			return fmt.Errorf("build tracer for %v error, %s", proto, err)
		}

		holder.Tracer = tracer
	}

	return nil
}

func (d *jaegerDriver) Register(proto types.ProtocolName, builder api.TracerBuilder) {
	d.tracers[proto] = &holder{
		TracerBuilder: builder,
	}
}

func (d *jaegerDriver) Get(proto types.ProtocolName) api.Tracer {
	if holder, ok := d.tracers[proto]; ok {
		return holder.Tracer
	}
	return nil
}

// NewJaegerImpl create jaeger driver
func NewJaegerImpl() api.Driver {
	return &jaegerDriver{
		tracers: make(map[types.ProtocolName]*holder),
	}
}
