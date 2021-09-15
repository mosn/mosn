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
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	monkey "github.com/cch123/supermonkey"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/upstream/cluster"
	_ "mosn.io/mosn/pkg/upstream/cluster"
	registry "mosn.io/pkg/registry/dubbo"
	dubbocommon "mosn.io/pkg/registry/dubbo/common"
	dubboconsts "mosn.io/pkg/registry/dubbo/common/constant"
	"mosn.io/pkg/registry/dubbo/remoting"
	zkreg "mosn.io/pkg/registry/dubbo/zookeeper"
)

func init() {
	initAPI()

	monkey.Patch(zkreg.NewZkRegistry, func(url *dubbocommon.URL) (registry.Registry, error) {
		return &registry.BaseRegistry{}, nil
	})

	monkey.PatchInstanceMethod(reflect.TypeOf(&registry.BaseRegistry{}), "Register", func(r *registry.BaseRegistry, conf *dubbocommon.URL) error {
		return nil
	})

	monkey.PatchInstanceMethod(reflect.TypeOf(&registry.BaseRegistry{}), "Subscribe", func(r *registry.BaseRegistry, url *dubbocommon.URL, notifyListener registry.NotifyListener) error {
		// do nothing
		return nil
	})

	cluster.NewClusterManagerSingleton(nil, nil, nil)

	Init(12356, 13245, "./www.log")
}

func TestNotify(t *testing.T) {
	// notify provider addition
	// the cluster should have 1 host
	var l = listener{}
	var event = registry.ServiceEvent{
		Action: remoting.EventTypeAdd,
		Service: *dubbocommon.NewURLWithOptions(
			dubbocommon.WithParams(url.Values{
				dubboconsts.INTERFACE_KEY: []string{"com.mosn.test.UserProvider"},
			},
			),
		)}

	l.Notify(&event)

	event = registry.ServiceEvent{
		Action: remoting.EventTypeDel,
		Service: *dubbocommon.NewURLWithOptions(
			dubbocommon.WithParams(url.Values{
				dubboconsts.INTERFACE_KEY: []string{"com.mosn.test.UserProvider"},
			},
			),
		)}

	l.Notify(&event)

	event = registry.ServiceEvent{
		Action: remoting.EventTypeUpdate,
		Service: *dubbocommon.NewURLWithOptions(
			dubbocommon.WithParams(url.Values{
				dubboconsts.INTERFACE_KEY: []string{"com.mosn.test.UserProvider"},
			},
			),
		)}

	l.Notify(&event)
}

func TestPubBindFail(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/pub", publish)

	req, _ := http.NewRequest("POST", "/pub", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	var res resp
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.Nil(t, err)
	assert.Equal(t, res.Errno, fail)
}

func TestPub(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/pub", publish)

	req, _ := http.NewRequest("POST", "/pub", strings.NewReader(`
	{
		"registry": {
			"addr": "127.0.0.1:2181",
			"type": "zookeeper"
		},
		"service": {
			"group": "",
			"interface": "com.ikurento.user",
			"methods": [
				"GetUser",
				"GetProfile",
				"kkk"
			],
			"name": "UserProvider",
			"port": "20000",
			"version": ""
		}
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	var res resp
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.Nil(t, err)
	assert.Equal(t, res.Errno, succ)
}

func TestSubBindFail(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/sub", subscribe)

	req, _ := http.NewRequest("POST", "/sub", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	var res resp
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.Nil(t, err)
	assert.Equal(t, res.Errno, fail)
}

func TestSub(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/sub", subscribe)

	req, _ := http.NewRequest("POST", "/sub", strings.NewReader(`
	{
		"registry": {
			"addr": "127.0.0.1:2181",
			"type": "zookeeper"
		},
		"service": {
			"group": "",
			"interface": "com.ikurento.user",
			"methods": [
				"GetUser"
			],
			"name": "UserProvider"
		}
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	var res resp
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.Nil(t, err)
	assert.Equal(t, res.Errno, succ)
}

func TestUnsub(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/unsub", unsubscribe)

	req, _ := http.NewRequest("POST", "/unsub", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)
}

func TestUnpub(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/unpub", unpublish)

	req, _ := http.NewRequest("POST", "/unpub", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)
}
