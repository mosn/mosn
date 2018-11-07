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

// "routersMap" in "routersMangerInstance" stored all routers with "RouterConfigureName" as the unique identifier

// when update, update wrapper's routes

// when use, proxy's get wrapper's routers

package router

import (
	"encoding/json"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

var routerConfig = `{
		"router_config_name":"test_router",
		"virtual_hosts": [
			{
				"name": "test_virtual_host1",
				"domains":["*"],
				"virtual_clusters":[
					{
						"name":"vc",
						"pattern":"test"
					}
				],
				"routers":[
					{
						"match": {
							"prefix":"/",
							"runtime": {
								"default_value":10,
								"runtime_key":"test"
							},
							"headers":[
								{
									"name":"service",
									"value":"test"
								}
							]
						},
						"route":{
							"cluster_name":"cluster",
							"weighted_clusters": [
								{
									"cluster": {
										"name": "test",
										"weight":100,
										"metadata_match": {
											"filter_metadata": {
												"mosn.lb": {
													"test":"test"
												}
											}
										}
									}
								}
							],
							"metadata_match": {
								"filter_metadata": {
									"mosn.lb": {
										"test":"test"
									}
								}
							},
							"timeout": "1s",
							"retry_policy":{
								"retry_on": true,
								"retry_timeout": "1m",
								"num_retries":10
							}
						},
						"redirect":{
							"host_redirect": "test",
							"response_code": 302
						},
						"metadata":{
							"filter_metadata": {
								"mosn.lb": {
									 "test":"test"
								}
							}
						},
						"decorator":"test"
					}
				]
			}
		]
	}`

var routerConfig2 = `{
		"router_config_name":"test_router",
		"virtual_hosts": [
			{
				"name": "test_virtual_host2",
				"domains":["www.antfin.com"],
				"virtual_clusters":[
					{
						"name":"vc",
						"pattern":"test"
					}
				],
				"routers":[
					{
						"match": {
							"prefix":"/",
							"runtime": {
								"default_value":10,
								"runtime_key":"test"
							},
							"headers":[
								{
									"name":"service",
									"value":"test"
								}
							]
						},
						"route":{
							"cluster_name":"cluster",
							"weighted_clusters": [
								{
									"cluster": {
										"name": "test",
										"weight":100,
										"metadata_match": {
											"filter_metadata": {
												"mosn.lb": {
													"test":"test"
												}
											}
										}
									}
								}
							],
							"metadata_match": {
								"filter_metadata": {
									"mosn.lb": {
										"test":"test"
									}
								}
							},
							"timeout": "1s",
							"retry_policy":{
								"retry_on": true,
								"retry_timeout": "1m",
								"num_retries":10
							}
						},
						"redirect":{
							"host_redirect": "test",
							"response_code": 302
						},
						"metadata":{
							"filter_metadata": {
								"mosn.lb": {
									 "test":"test"
								}
							}
						},
						"decorator":"test"
					}
				]
			}
		]
	}`

func Test_NewRouterManager(t *testing.T) {
	routerManager := NewRouterManager()
	if routerManager == nil || routerManager != GetRoutersMangerInstance() {
		t.Errorf("new router manager error")
	}
}

func Test_GetRoutersMangerInstance(t *testing.T) {
	routerManager := NewRouterManager()
	if routerManager == nil || routerManager != GetRoutersMangerInstance() {
		t.Errorf("get router manager error")
	}
}

func Test_routersManager_AddOrUpdateRouters(t *testing.T) {
	routerManager := NewRouterManager()
	bytes := []byte(routerConfig)
	router := &v2.RouterConfiguration{}

	if err := json.Unmarshal(bytes, router); err != nil {
		t.Errorf(err.Error())
	}

	routerConfigName := "test_router"

	if _, ok := GetRoutersMangerInstance().routersMap.Load(routerConfigName); ok {
		t.Errorf("test_router already exist")
	}

	if err := routerManager.AddOrUpdateRouters(router); err != nil {
		t.Errorf(err.Error())
	} else {
		if value, ok := GetRoutersMangerInstance().routersMap.Load(routerConfigName); !ok {
			t.Errorf("AddOrUpdateRouters error, %s not found", routerConfigName)
		} else {
			if primaryRouters, ok := value.(*RoutersWrapper); ok {
				routerMatcher := primaryRouters.routers.(*routeMatcher)
				if routerMatcher.defaultVirtualHost == nil || routerMatcher.defaultVirtualHost.Name() != "test_virtual_host1" {
					t.Error("AddOrUpdateRouters error")
				}
			}
		}
	}
}

func Test_routersManager_GetRouterWrapperByName(t *testing.T) {

	bytes1 := []byte(routerConfig)
	router1 := &v2.RouterConfiguration{}

	bytes2 := []byte(routerConfig2)
	router2 := &v2.RouterConfiguration{}

	routerConfigName := "test_router"

	router0 := &v2.RouterConfiguration{
		RouterConfigName: routerConfigName,
	}

	if err := json.Unmarshal(bytes1, router1); err != nil {
		t.Errorf(err.Error())
	}

	if err := json.Unmarshal(bytes2, router2); err != nil {
		t.Errorf(err.Error())
	}

	routerManager := NewRouterManager()
	routerManager.AddOrUpdateRouters(router0)
	routeWrapper0 := routerManager.GetRouterWrapperByName(routerConfigName)
	routers0 := routeWrapper0.GetRouters()

	// add routers1 to "test_router"
	routerManager.AddOrUpdateRouters(router1)
	routerWrapper1 := routerManager.GetRouterWrapperByName(routerConfigName)
	routers1 := routerWrapper1.GetRouters()

	// update "test_router" with router2
	routerManager.AddOrUpdateRouters(router2)
	routerWrapper2 := routerManager.GetRouterWrapperByName(routerConfigName)
	routers2 := routerWrapper2.GetRouters()

	routers0_ := routeWrapper0.GetRouters()
	routers1_ := routerWrapper1.GetRouters()

	// expect routers has been updated
	if routers0 == routers1 || routers1 == routers2 {
		t.Error("expect routers has been updated but not")
	}

	// expect wrapper still the same
	if routeWrapper0 != routerWrapper1 || routerWrapper1 != routerWrapper2 {
		t.Error("expect wrapper still the same but not")
	}

	// expect router has been updated for origin wrapper
	if routers0_ != routers2 || routers1_ != routers2 {
		t.Error("expect wrapper still the same but not ")
	}
}
