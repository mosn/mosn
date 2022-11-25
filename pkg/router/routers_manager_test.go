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
	"context"
	"testing"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
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

	if _, ok := routersManagerInstance.routersWrapperMap.Load(routerConfigName); ok {
		t.Errorf("test_router already exist")
	}

	if err := routerManager.AddOrUpdateRouters(router); err != nil {
		t.Errorf(err.Error())
	} else {
		if value, ok := routersManagerInstance.routersWrapperMap.Load(routerConfigName); !ok {
			t.Errorf("AddOrUpdateRouters error, %s not found", routerConfigName)
		} else {
			if primaryRouters, ok := value.(*RoutersWrapper); ok {
				routerMatcher := primaryRouters.routers.(*routersImpl)
				if routerMatcher.defaultVirtualHostIndex == -1 {
					t.Error("AddOrUpdateRouters error")
				} else if routerMatcher.virtualHosts[routerMatcher.defaultVirtualHostIndex].Name() != "test_virtual_host1" {
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
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: routerConfigName,
		},
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

func Test_routersManager_AddRouter(t *testing.T) {
	routerManager := NewRouterManager()
	routerCfg := &v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: "test_addrouter",
		},
		VirtualHosts: []v2.VirtualHost{
			{
				Name:    "test_addrouter_vh",
				Domains: []string{"www.test.com"},
				// no touters
			},
			{
				Name:    "test_default",
				Domains: []string{"*"},
			},
		},
	}
	if err := routerManager.AddOrUpdateRouters(routerCfg); err != nil {
		t.Fatal("init router config failed")
	}
	rw := routerManager.GetRouterWrapperByName("test_addrouter")
	if rw == nil {
		t.Fatal("can not find router wrapper")
	}
	// test add router
	routeCfg := &v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{
				Headers: []v2.HeaderMatcher{
					{
						Name:  "service",
						Value: "test",
					},
				},
			},
		},
	}
	if err := routerManager.AddRoute("test_addrouter", "www.test.com", routeCfg); err != nil {
		t.Fatal("add router failed", err)
	}
	ctx := variable.NewVariableContext(context.Background())
	routers := rw.GetRouters()
	// the wrapper can get the new router
	variable.SetString(ctx, types.VarHost, "www.test.com")
	if r := routers.MatchRouteFromHeaderKV(ctx, nil, "service", "test"); r == nil {
		t.Fatal("added route, but can not find it")
	}
	variable.SetString(ctx, types.VarHost, "www.test.net")
	if r := routers.MatchRouteFromHeaderKV(ctx, nil, "service", "test"); r != nil {
		t.Fatal("not added route, but still find it")
	}
	// test config is expected changed
	cfg := rw.GetRoutersConfig()
	if len(cfg.VirtualHosts[0].Routers) != 1 || len(cfg.VirtualHosts[1].Routers) != 0 {
		t.Fatal("route config is not changed")
	}

	// test add into default
	if err := routerManager.AddRoute("test_addrouter", "", routeCfg); err != nil {
		t.Fatal("add router failed", err)
	}
	// config is changed
	cfgChanged := rw.GetRoutersConfig()
	if len(cfgChanged.VirtualHosts[0].Routers) != 1 || len(cfgChanged.VirtualHosts[1].Routers) != 1 {
		t.Fatal("default route config is not changed")
	}
	if len(cfgChanged.VirtualHosts[0].Routers[0].Match.Headers) != 1 {
		t.Fatal("virtual host config routers is not expected")
	}
	routersChanged := rw.GetRouters()
	// the wrapper can get the new router
	if r := routersChanged.MatchRouteFromHeaderKV(ctx, nil, "service", "test"); r == nil {
		t.Fatal("added route, but can not find it")
	}
}

func Test_routersManager_RemoveAllRouter(t *testing.T) {
	ctx := variable.NewVariableContext(context.Background())
	routerManager := NewRouterManager()
	routerCfg := &v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: "test_remove_all_router",
		},
		VirtualHosts: []v2.VirtualHost{
			{
				Name:    "test_addrouter_vh",
				Domains: []string{"www.test.com"},
				Routers: []v2.Router{
					{
						RouterConfig: v2.RouterConfig{
							Match: v2.RouterMatch{
								Headers: []v2.HeaderMatcher{
									{
										Name:  "service",
										Value: "test",
									},
								},
							},
						},
					},
				},
			},
			{
				Name:    "test_default",
				Domains: []string{"*"},
				Routers: []v2.Router{
					{
						RouterConfig: v2.RouterConfig{
							Match: v2.RouterMatch{
								Headers: []v2.HeaderMatcher{
									{
										Name:  "service",
										Value: "test",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	// init
	if err := routerManager.AddOrUpdateRouters(routerCfg); err != nil {
		t.Fatal("init router config failed")
	}
	rw := routerManager.GetRouterWrapperByName("test_remove_all_router")
	if rw == nil {
		t.Fatal("can not find router wrapper")
	}
	// remove
	if err := routerManager.RemoveAllRoutes("test_remove_all_router", "www.test.com"); err != nil {
		t.Fatal("remove all router failed", err)
	}
	routers := rw.GetRouters()
	variable.SetString(ctx, types.VarHost, "www.test.com")
	if r := routers.MatchRouteFromHeaderKV(ctx, nil, "service", "test"); r != nil {
		t.Fatal("remove route, but still can matched")
	}
	ctx2 := variable.NewVariableContext(context.Background())
	variable.SetString(ctx2, types.VarHost, "www.test.net")
	if r := routers.MatchRouteFromHeaderKV(ctx2, nil, "service", "test"); r == nil {
		t.Fatal("route removed unexpected")
	}
	// test config is expected changed
	cfg := rw.GetRoutersConfig()
	if len(cfg.VirtualHosts[0].Routers) != 0 || len(cfg.VirtualHosts[1].Routers) != 1 {
		t.Fatal("route config is not changed")
	}

	// test remove default
	if err := routerManager.RemoveAllRoutes("test_remove_all_router", ""); err != nil {
		t.Fatal("remove router failed", err)
	}
	// config is changed
	cfgChanged := rw.GetRoutersConfig()
	if len(cfgChanged.VirtualHosts[0].Routers) != 0 || len(cfgChanged.VirtualHosts[1].Routers) != 0 {
		t.Fatal("default route config is not changed")
	}
	routersChanged := rw.GetRouters()
	// the wrapper can get the new router
	if r := routersChanged.MatchRouteFromHeaderKV(ctx2, nil, "service", "test"); r != nil {
		t.Fatal("remove route, but still can matched")
	}
}
