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
	"fmt"
	"net/http"
	"net/url"

	"mosn.io/mosn/pkg/trace"
	dubbocommon "mosn.io/pkg/registry/dubbo/common"
	dubboconsts "mosn.io/pkg/registry/dubbo/common/constant"
)

var mosnPubPort = "33333" // default

// publish a service to registry
func publish(w http.ResponseWriter, r *http.Request) {
	var req pubReq

	err := bind(r, &req)
	if err != nil {
		response(w, resp{Errno: fail, ErrMsg: err.Error()})
		return
	}

	err = doPubUnPub(req, true)
	if err != nil {
		response(w, resp{Errno: fail, ErrMsg: "publish fail, err: " + err.Error()})
		return
	}

	response(w, resp{Errno: succ, ErrMsg: "publish success"})
	return
}

// unpublish user service from registry
func unpublish(w http.ResponseWriter, r *http.Request) {
	var req pubReq
	err := bind(r, &req)
	if err != nil {
		response(w, resp{Errno: fail, ErrMsg: err.Error()})
		return
	}

	err = doPubUnPub(req, false)
	if err != nil {
		response(w, resp{Errno: fail, ErrMsg: "unpub fail, err: " + err.Error()})
		return
	}

	response(w, resp{Errno: succ, ErrMsg: "unpub success"})
	return

}

func doPubUnPub(req pubReq, pub bool) error {
	var registryPath = registryPathTpl.ExecuteString(map[string]interface{}{
		"addr": req.Registry.Addr,
	})

	registryURL, err := dubbocommon.NewURL(registryPath,
		dubbocommon.WithParams(url.Values{
			dubboconsts.REGISTRY_KEY:         []string{req.Registry.Type},
			dubboconsts.REGISTRY_TIMEOUT_KEY: []string{"5s"},
			dubboconsts.ROLE_KEY:             []string{fmt.Sprint(dubbocommon.PROVIDER)},
		}),
		dubbocommon.WithUsername(req.Registry.UserName),
		dubbocommon.WithPassword(req.Registry.Password),
		dubbocommon.WithLocation(req.Registry.Addr),
	)
	if err != nil {
		return err
	}

	// find registry from cache
	registryCacheKey := req.Service.Interface
	reg, err := getRegistry(registryCacheKey, dubbocommon.PROVIDER, &registryURL)
	if err != nil {
		return err
	}

	var dubboPath = dubboPathTpl.ExecuteString(map[string]interface{}{
		"ip":        trace.GetIp(),
		"port":      mosnPubPort,
		"interface": req.Service.Interface,
	})
	urlValues := url.Values{
		dubboconsts.ROLE_KEY:      []string{fmt.Sprint(dubbocommon.PROVIDER)},
		dubboconsts.INTERFACE_KEY: []string{req.Service.Interface},
	}
	if req.Service.Group != "" {
		urlValues.Set(dubboconsts.GROUP_KEY, req.Service.Group)
	}
	if req.Service.Version != "" {
		urlValues.Set(dubboconsts.VERSION_KEY, req.Service.Version)
	}
	dubboURL, _ := dubbocommon.NewURL(dubboPath,
		dubbocommon.WithParams(urlValues),
		dubbocommon.WithMethods(req.Service.Methods))

	if pub {
		// publish this service
		return reg.Register(&dubboURL)
	}

	// unpublish this service
	return reg.UnRegister(&dubboURL)

}
