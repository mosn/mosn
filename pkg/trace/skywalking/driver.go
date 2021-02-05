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
	"fmt"

	"mosn.io/api"

	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
)

const (
	MOSNComponentID = 5003
	ErrorLog        = "Errors on upstream service"

	SkyDriverName = "SkyWalking"
)

func init() {
	trace.RegisterDriver(SkyDriverName, NewSkyDriverImpl())
}

type holder struct {
	api.Tracer
	api.TracerBuilder
}

type skyDriver struct {
	tracers map[types.ProtocolName]*holder
}

func (d *skyDriver) Init(config map[string]interface{}) error {
	// create go2sky reporter
	t, err := newGO2SkyTracer(config)
	if err != nil {
		return err
	}
	for proto, holder := range d.tracers {
		tracer, err := holder.TracerBuilder(config)
		if err != nil {
			return fmt.Errorf("build tracer for %v error, %s", proto, err)
		}
		if skyTracer, ok := tracer.(SkyTracer); ok {
			// injection go2sky.Tracer
			skyTracer.SetGO2SkyTracer(t)
		}
		holder.Tracer = tracer
	}
	return nil
}

func (d *skyDriver) Register(proto types.ProtocolName, builder api.TracerBuilder) {
	d.tracers[proto] = &holder{
		TracerBuilder: builder,
	}
}

func (d *skyDriver) Get(proto types.ProtocolName) api.Tracer {
	if holder, ok := d.tracers[proto]; ok {
		return holder.Tracer
	}
	return nil
}

func NewSkyDriverImpl() api.Driver {
	return &skyDriver{
		tracers: make(map[types.ProtocolName]*holder),
	}
}
