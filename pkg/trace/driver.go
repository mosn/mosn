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
	"fmt"

	"mosn.io/mosn/pkg/types"
)

type holder struct {
	types.Tracer
	types.TracerBuilder
}

type defaultDriver struct {
	tracers map[types.ProtocolName]*holder
}

func (d *defaultDriver) Init(config map[string]interface{}) error {
	for proto, holder := range d.tracers {
		tracer, err := holder.TracerBuilder(config)
		if err != nil {
			return fmt.Errorf("build tracer for %v error, %s", proto, err)
		}
		holder.Tracer = tracer
	}
	return nil
}

func (d *defaultDriver) Register(proto types.ProtocolName, builder types.TracerBuilder) {
	d.tracers[proto] = &holder{
		TracerBuilder: builder,
	}
}

func (d *defaultDriver) Get(proto types.ProtocolName) types.Tracer {
	if holder, ok := d.tracers[proto]; ok {
		return holder.Tracer
	}
	return nil
}

func NewDefaultDriverImpl() types.Driver {
	return &defaultDriver{
		tracers: make(map[types.ProtocolName]*holder),
	}
}
