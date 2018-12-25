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

package config

import (
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

func mockInitConfig(t *testing.T, cfg []byte) {
	// config is a global var
	if err := json.Unmarshal(cfg, &config); err != nil {
		t.Fatal("init config failed", err)
	}
}

func TestAddRoutersConfig(t *testing.T) {
	// only keep useful test part
	cfg := []byte(`{
		"servers": [
			{
				"listeners": [
					{
						"name": "egress",
						"filter_chains": [
							{
								"filters": [
									{
										"type":"connection_manager",
										"config":{
											"router_config_name":"egress_router",
											"virtual_hosts":[
												{
													"name": "egress",
													"domains":["*"],
													"routers": [
														{
															"match": {"prefix":"/"},
															"route":{"cluster_name":"test1"}
														}
													]
												}
											]
										}		
									}	
								]
							}
						]
					},
					{
						"name": "ingress",
						"filter_chains": [
							{
								"filters": [
									{
										"type":"connection_manager",
									 	"config":{
											"router_config_name":"egress_router",
										 	"virtual_hosts":[
										 		{
													"name": "ingress",
													"domains":["*"],
													"routers": [
														{
															"match": {"prefix":"/"},
															"route":{"cluster_name":"test1"}
														}
													]
												}
										 	]
									 	}
									}
								]
							}
						]
					}
				]
			}
		]
	}`)
	mockInitConfig(t, cfg)
	router := v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{
				Headers: []v2.HeaderMatcher{
					{
						Name:  "service",
						Value: "test_new",
					},
				},
			},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: "test_new",
				},
			},
		},
	}
	// add a default virtual host
	if ok := addRoutersConfig("egress", "", router); !ok {
		t.Error("add egress router failed")
	}
	// add a specify virtual host
	if ok := addRoutersConfig("ingress", "ingress", router); !ok {
		t.Error("add ingress router failed")
	}
	// add a listener not exists
	if ok := addRoutersConfig("not_exists", "", router); ok {
		t.Error("add a not exists listener config")
	}
	// verify
	listeners := config.Servers[0].Listeners
	for _, ln := range listeners {
		filter := ln.FilterChains[0].Filters[0] // only one connection_manager
		if filter.Type != v2.CONNECTION_MANAGER {
			t.Errorf("listener %s filter is not expected, get %s", ln.Name, filter.Type)
			continue
		}
		routerConfiguration := &v2.RouterConfiguration{}
		if data, err := json.Marshal(filter.Config); err == nil {
			if err := json.Unmarshal(data, routerConfiguration); err != nil {
				t.Errorf("invalid router config got, listener %s", ln.Name)
				continue
			}
		}
		if len(routerConfiguration.VirtualHosts[0].Routers) != 2 {
			t.Errorf("not enough router rule found in listener %s", ln.Name)
		}
		findOriginal := false
		findNew := false
		for _, r := range routerConfiguration.VirtualHosts[0].Routers {
			switch r.Route.ClusterName {
			case "test_new":
				findNew = true
			case "test1":
				findOriginal = true
			}
		}
		if !findOriginal || !findNew {
			t.Errorf("listener %s, original router: %v, new router: %v", ln.Name, findOriginal, findNew)
		}
	}
}
