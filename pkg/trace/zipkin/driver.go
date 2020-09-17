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

	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
)

func init() {
	trace.RegisterDriver(ZipkinTracer, NewZipkinDriverImpl())
}

type holder struct {
	types.Tracer
	types.TracerBuilder
}

type zipkinDriver struct {
	tracers map[types.ProtocolName]*holder
}

func (d *zipkinDriver) Init(config map[string]interface{}) error {
	t, err := newGO2ZipkinTracer(config)
	if err != nil {
		return err
	}
	for proto, holder := range d.tracers {
		tracer, err := holder.TracerBuilder(config)
		if err != nil {
			return fmt.Errorf("build tracer for %v error, %s", proto, err)
		}
		if zipkinTracer, ok := tracer.(zipkinTracer); ok {
			// injection zipkin.Tracer
			zipkinTracer.SetGO2ZipkinTracer(t)
		}
		holder.Tracer = tracer
	}
	return nil
}

func (d *zipkinDriver) Register(proto types.ProtocolName, builder types.TracerBuilder) {
	d.tracers[proto] = &holder{
		TracerBuilder: builder,
	}
}

func (d *zipkinDriver) Get(proto types.ProtocolName) types.Tracer {
	if holder, ok := d.tracers[proto]; ok {
		return holder.Tracer
	}
	return nil
}

func NewZipkinDriverImpl() types.Driver {
	return &zipkinDriver{
		tracers: make(map[types.ProtocolName]*holder),
	}
}
