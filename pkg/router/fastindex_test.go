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
	"testing"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

func createRoutersCfg(cnt int) []v2.Router {
	routersCfg := []v2.Router{}
	for i := 0; i < cnt; i++ {
		r := v2.Router{
			RouterConfig: v2.RouterConfig{
				Match: v2.RouterMatch{
					Headers: []v2.HeaderMatcher{
						{
							Name:  types.SofaRouteMatchKey,
							Value: fmt.Sprintf("service#%d", i),
						},
					},
				},
				Route: v2.RouteAction{
					RouterActionConfig: v2.RouterActionConfig{
						ClusterName: fmt.Sprintf("cluster#%d", i),
					},
				},
			},
		}
		routersCfg = append(routersCfg, r)
	}
	return routersCfg
}

// we configured some sofa rpc routers without regex(.*)
// without fast index, we  iterate through the slice and  find the first matching route.
// with the fast index, we get the route directly from the key&value
func TestFastIndexRouteFromHeaderKV(t *testing.T) {
	routersCnt := 10
	routersCfg := createRoutersCfg(routersCnt)
	vhCfg := &v2.VirtualHost{
		Domains: []string{"*"},
		Routers: routersCfg,
	}
	vh, err := NewVirtualHostImpl(vhCfg)
	if err != nil {
		t.Fatal("create virtual host failed", err)
	}
	for i := 0; i < routersCnt; i++ {
		value := fmt.Sprintf("service#%d", i)
		expected := fmt.Sprintf("cluster#%d", i)
		if route := vh.GetRouteFromHeaderKV(types.SofaRouteMatchKey, value); route == nil || route.RouteRule().ClusterName() != expected {
			t.Errorf("#%d route match is not expected, route: %v", i, route)
		}
	}
}

func TestMatchRouteFromHeaderKV(t *testing.T) {
	routersCnt := 10
	routersCfg := createRoutersCfg(routersCnt)
	vhCfg := &v2.VirtualHost{
		Domains: []string{"*"},
		Routers: routersCfg,
	}
	routeCfg := &v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: "test",
		},
		VirtualHosts: []*v2.VirtualHost{
			vhCfg,
		},
	}
	routers, err := NewRouters(routeCfg)
	if err != nil {
		t.Fatal("create route matcher failed")
	}
	for i := 0; i < routersCnt; i++ {
		value := fmt.Sprintf("service#%d", i)
		expected := fmt.Sprintf("cluster#%d", i)
		// use header to find virtual host, in this case only have default virtualhost, header can be nil
		if route := routers.MatchRouteFromHeaderKV(nil, types.SofaRouteMatchKey, value); route == nil || route.RouteRule().ClusterName() != expected {
			t.Errorf("#%d route match is not expected, route: %v", i, route)
		}
	}

}

func BenchmarkGetSofaRouter(b *testing.B) {
	log.DefaultLogger.SetLogLevel(log.FATAL)

	routersCnt := 5000
	routersCfg := createRoutersCfg(routersCnt)
	vhCfg := &v2.VirtualHost{
		Domains: []string{"*"},
		Routers: routersCfg,
	}
	vh, err := NewVirtualHostImpl(vhCfg)
	if err != nil {
		b.Fatal("create virtual host failed", err)
	}
	value := fmt.Sprintf("service#%d", 3000)
	expected := fmt.Sprintf("cluster#%d", 3000)
	b.Run("kv", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if route := vh.GetRouteFromHeaderKV(types.SofaRouteMatchKey, value); route == nil || route.RouteRule().ClusterName() != expected {
				b.Errorf("route match is not expected, route: %v", route)
			}
		}
	})
	headers := protocol.CommonHeader(map[string]string{
		types.SofaRouteMatchKey: value,
	})
	b.Run("common", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if route := vh.GetRouteFromEntries(headers, 1); route == nil || route.RouteRule().ClusterName() != expected {
				b.Errorf("route match is not expected, route: %v", route)
			}
		}
	})
}
