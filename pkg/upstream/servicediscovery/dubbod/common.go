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
	"math/rand"
	"net/http"
	"sync"

	"github.com/mosn/binding"
	dubbocommon "github.com/mosn/registry/dubbo/common"
	zkreg "github.com/mosn/registry/dubbo/zookeeper"
	dubboreg "github.com/mosn/registry/dubbo"
	"github.com/valyala/fasttemplate"
)

var (
	dubboPathTpl    = fasttemplate.New("dubbo://{{ip}}:{{port}}/{{interface}}", "{{", "}}")
	registryPathTpl = fasttemplate.New("registry://{{addr}}", "{{", "}}")
	dubboRouterConfigName = "dubbo" // keep the same with the router config name in mosn_config.json
)

var (
	mosnIP, mosnPort = "127.0.0.1", fmt.Sprint(rand.Int63n(30000)+1) // TODO, need to read from mosn config
)

const (
	succ = iota
	fail
)

// /com.test.cch.UserService --> zk client
var registryClientCache = sync.Map{}

func getRegistry(registryCacheKey string, role int, registryURL dubbocommon.URL) (dubboreg.Registry, error) {
	// do not cache provider registry, or it may collide with the consumer registry
	if role == dubbocommon.PROVIDER {
		return zkreg.NewZkRegistry(&registryURL)
	}

	regInterface, ok := registryClientCache.Load(registryCacheKey)

	var (
		reg dubboreg.Registry
		err error
	)

	if !ok {
		// init registry
		reg, err = zkreg.NewZkRegistry(&registryURL)
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

