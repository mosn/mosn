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
	"plugin"
	"strings"

	"mosn.io/api"

	"mosn.io/mosn/pkg/types"
)

type holder struct {
	api.Tracer
	api.TracerBuilder
}

type defaultDriver struct {
	tracers map[types.ProtocolName]*holder
}

func (d *defaultDriver) Init(config map[string]interface{}) error {
	tracers, err := d.loadPlugin(config)
	if err != nil {
		return err
	}
	for proto, holder := range d.tracers {
		if tracer, ok := tracers[proto]; ok {
			holder.Tracer = tracer
			continue
		}
		tracer, err := holder.TracerBuilder(config)
		if err != nil {
			return fmt.Errorf("build tracer for %v error, %s", proto, err)
		}
		holder.Tracer = tracer
	}
	return nil
}

func (d *defaultDriver) loadPlugin(config map[string]interface{}) (map[api.ProtocolName]api.Tracer, error) {
	tracers := make(map[api.ProtocolName]api.Tracer)
	ps, ok1 := config["protocols"].(string)
	sopath, ok2 := config["sopath"].(string)
	if !ok1 || !ok2 {
		return tracers, nil
	}

	loaderFuncName, ok := config["factory_method"].(string)
	if !ok {
		loaderFuncName = "TracerBuilder"
	}

	p, err := plugin.Open(sopath)
	if err != nil {
		return nil, err
	}

	sym, err := p.Lookup(loaderFuncName)
	if err != nil {
		return nil, err
	}

	loadFunc, ok := sym.(func(config map[string]interface{}) (api.Tracer, error))
	if !ok {
		return nil, err
	}

	for _, proto := range strings.Split(ps, ",") {
		tracer, err := loadFunc(config)
		if err != nil {
			return nil, fmt.Errorf("build tracer for %v error, %s", proto, err)
		}
		tracers[api.ProtocolName(proto)] = tracer
	}
	return tracers, nil
}

func (d *defaultDriver) Register(proto types.ProtocolName, builder api.TracerBuilder) {
	d.tracers[proto] = &holder{
		TracerBuilder: builder,
	}
}

func (d *defaultDriver) Get(proto types.ProtocolName) api.Tracer {
	if holder, ok := d.tracers[proto]; ok {
		return holder.Tracer
	}
	return nil
}

func NewDefaultDriverImpl() api.Driver {
	return &defaultDriver{
		tracers: make(map[types.ProtocolName]*holder),
	}
}
