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

package router

import (
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/types"
)

type configFactory func(config interface{}) (types.Routers, error)

var routerConfigFactories map[types.Protocol]configFactory

// RegisterRouterConfigFactory
// register router config factory for protocol
func RegisterRouterConfigFactory(port types.Protocol, factory configFactory) {
	if routerConfigFactories == nil {
		routerConfigFactories = make(map[types.Protocol]configFactory)
	}

	if _, ok := routerConfigFactories[port]; !ok {
		routerConfigFactories[port] = factory
	}
}

// CreateRouteConfig
// return route factory according to protocol as input
func CreateRouteConfig(port types.Protocol, config interface{}) (types.Routers, error) {
	if factory, ok := routerConfigFactories[port]; ok {
		return factory(config) //call NewBasicRoute
	}

	return nil, fmt.Errorf("Unsupported protocol %s", port)
}
