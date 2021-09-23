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

package mtls

import (
	"fmt"

	"mosn.io/mosn/pkg/mtls/sds"
	"mosn.io/mosn/pkg/types"
)

const defaultFactoryName = ""

var factories map[string]ConfigHooksFactory

func init() {
	factories = map[string]ConfigHooksFactory{
		defaultFactoryName: &defaultFactory{},
	}
}

// Register registers an extension.
func Register(name string, factory ConfigHooksFactory) error {
	if _, ok := factories[name]; ok {
		return fmt.Errorf("%s extesions is already registered", name)
	}
	factories[name] = factory
	return nil
}

func getFactory(name string) ConfigHooksFactory {
	if factory, ok := factories[name]; ok {
		return factory
	}
	return factories[defaultFactoryName]
}

type defaultFactory struct{}

func (f *defaultFactory) CreateConfigHooks(config map[string]interface{}) ConfigHooks {
	return &defaultConfigHooks{}
}

// for test
var getSdsClientFunc func(cfg interface{}) types.SdsClient = sds.NewSdsClientSingleton

func GetSdsClient(cfg interface{}) types.SdsClient {
	return getSdsClientFunc(cfg)
}
