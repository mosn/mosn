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
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

func init() {
	RegisterMakeHandler(types.DefaultRouteHandler, DefaultMakeHandler, true)
}

var makeHandler = &handlerFactories{
	factories: map[string]MakeHandlerFunc{},
}

func RegisterMakeHandler(name string, f MakeHandlerFunc, isDefault bool) {
	log.DefaultLogger.Infof("register a new handler maker, name is %s, is default: %t", name, isDefault)
	makeHandler.add(name, f, isDefault)
}

func GetMakeHandlerFunc(name string) MakeHandlerFunc {
	return makeHandler.get(name)
}

func MakeHandlerFuncExists(name string) bool {
	return makeHandler.exists(name)
}
