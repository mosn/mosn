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

	"mosn.io/api"
	"mosn.io/pkg/log"

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
	tracer, err := d.loadPlugin(config)
	if err != nil {
		log.DefaultLogger.Errorf("lood trace plugin failed err:%s", err.Error())
		return err
	}
	for proto, holder := range d.tracers {
		if tracer != nil {
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

func (d *defaultDriver) loadPlugin(config map[string]interface{}) (api.Tracer, error) {
	sopath, ok2 := config["sopath"].(string)
	if !ok2 {
		return nil, nil
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
	log.DefaultLogger.Infof("lood trace plugin funcname:%s,sopath:%s", loaderFuncName, sopath)
	return loadFunc(config)
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
