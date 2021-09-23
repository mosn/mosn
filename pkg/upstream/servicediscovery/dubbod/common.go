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
package dubbod

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	v2 "mosn.io/mosn/pkg/config/v2"
	routerAdapter "mosn.io/mosn/pkg/router"

	"github.com/valyala/fasttemplate"
	"mosn.io/pkg/binding"
	dubboreg "mosn.io/pkg/registry/dubbo"
	dubbocommon "mosn.io/pkg/registry/dubbo/common"
	zkreg "mosn.io/pkg/registry/dubbo/zookeeper"
)

var (
	dubboPathTpl          = fasttemplate.New("dubbo://{{ip}}:{{port}}/{{interface}}", "{{", "}}")
	registryPathTpl       = fasttemplate.New("registry://{{addr}}", "{{", "}}")
	dubboRouterConfigName = "dubbo" // keep the same with the router config name in mosn_config.json
)

const (
	succ = iota
	fail
)

// /com.test.cch.UserService --> zk client
var registryClientCache = sync.Map{}

func getRegistry(registryCacheKey string, role int, registryURL *dubbocommon.URL) (dubboreg.Registry, error) {

	registryCacheKey = registryCacheKey + "#" + fmt.Sprint(role)
	regInterface, ok := registryClientCache.Load(registryCacheKey)

	var (
		reg dubboreg.Registry
		err error
	)

	if !ok {
		// init registry
		reg, err = zkreg.NewZkRegistry(registryURL)
		// store registry object to global cache
		if err == nil {
			registryClientCache.Store(registryCacheKey, reg)
		}
	} else {
		reg = regInterface.(dubboreg.Registry)
	}

	return reg, err
}

func response(w http.ResponseWriter, respBody interface{}) {
	bodyBytes, err := json.Marshal(respBody)
	if err != nil {
		_, _ = w.Write([]byte("response marshal failed, err: " + err.Error()))
	}

	_, _ = w.Write(bodyBytes)
}

// bind the struct content from http.Request body/uri
func bind(r *http.Request, data interface{}) error {
	b := binding.Default(r.Method, r.Header.Get("Content-Type"))
	return b.Bind(r, data)
}

var dubboInterface2registerFlag = sync.Map{}

// add a router rule to router manager, avoid duplicate rules
func addRouteRule(servicePath string) error {
	// if already route rule of this service is already added to router manager
	// then skip
	if _, ok := dubboInterface2registerFlag.Load(servicePath); ok {
		return nil
	}

	dubboInterface2registerFlag.Store(servicePath, struct{}{})
	return routerAdapter.GetRoutersMangerInstance().AddRoute(dubboRouterConfigName, "*", &v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{
				Headers: []v2.HeaderMatcher{
					{
						Name:  "service", // use the xprotocol header field "service"
						Value: servicePath,
					},
				},
			},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: servicePath,
				},
			},
		},
	})
}
