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

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

func init() {
	RegisterRouterRule(DefaultSofaRouterRuleFactory, 1)
	RegisterMakeHandler(types.DefaultRouteHandler, DefaultMakeHandler, true)
}

var defaultRouterRuleFactoryOrder routerRuleFactoryOrder

func RegisterRouterRule(f RouterRuleFactory, order uint32) {
	if defaultRouterRuleFactoryOrder.order < order {
		log.DefaultLogger.Infof(RouterLogFormat, "Extend", "RegisterRouterRule", fmt.Sprintf("order is %d", order))
		defaultRouterRuleFactoryOrder.factory = f
		defaultRouterRuleFactoryOrder.order = order
	} else {
		msg := fmt.Sprintf("current register order is %d, order %d register failed", defaultRouterRuleFactoryOrder.order, order)
		log.DefaultLogger.Errorf(RouterLogFormat, "Extend", "RegisterRouterRule", msg)
	}
}

func DefaultSofaRouterRuleFactory(base *RouteRuleImplBase, headers []v2.HeaderMatcher) RouteBase {
	r := &SofaRouteRuleImpl{
		RouteRuleImplBase: base,
	}
	// compatible for simple sofa rule
	if len(headers) == 1 && headers[0].Name == types.SofaRouteMatchKey {
		r.fastmatch = headers[0].Value
	}
	return r
}

var makeHandler = &handlerFactories{
	factories: map[string]MakeHandlerFunc{},
}

func RegisterMakeHandler(name string, f MakeHandlerFunc, isDefault bool) {
	log.DefaultLogger.Infof("regist a new handler maker, name is %s, is default: %t", name, isDefault)
	makeHandler.add(name, f, isDefault)
}

func GetMakeHandlerFunc(name string) MakeHandlerFunc {
	return makeHandler.get(name)
}

func MakeHandlerFuncExists(name string) bool {
	return makeHandler.exists(name)
}
