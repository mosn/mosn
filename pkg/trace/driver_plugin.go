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
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
)

const (
	PluinDriverName = "PluginDriver"
)

func init() {
	RegisterDriver(PluinDriverName, NewPluginDriverImpl())
}

type pluginDriver struct {
	tracer api.Tracer
}

func (d *pluginDriver) Init(config map[string]interface{}) error {
	tracer, err := d.loadPlugin(config)
	if err != nil {
		log.DefaultLogger.Errorf("lood trace plugin failed err:%s", err.Error())
		return err
	}
	d.tracer = tracer
	return nil
}

func (d *pluginDriver) loadPlugin(config map[string]interface{}) (api.Tracer, error) {
	sopath, ok2 := config["sopath"].(string)
	if !ok2 || len(sopath) == 0 {
		return nil, fmt.Errorf("sopath is empty")
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

	builderc, ok := config["builder"].(map[string]interface{})
	if ok {
		return loadFunc(builderc)
	}
	return nil, fmt.Errorf("the builder config is empty")
}

func (d *pluginDriver) Register(proto types.ProtocolName, builder api.TracerBuilder) {
	log.DefaultLogger.Errorf("not support")
}

func (d *pluginDriver) Get(proto types.ProtocolName) api.Tracer {
	return d.tracer
}

func NewPluginDriverImpl() api.Driver {
	return &pluginDriver{}
}
