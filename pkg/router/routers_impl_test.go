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
	"context"
	"os"
	"testing"

	"mosn.io/pkg/variable"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

func newTestSimpleRouter(name string) v2.Router {
	r := v2.Router{}
	r.Match = v2.RouterMatch{
		Headers: []v2.HeaderMatcher{
			v2.HeaderMatcher{Name: "service", Value: ".*"},
		},
	}
	r.Route = v2.RouteAction{
		RouterActionConfig: v2.RouterActionConfig{
			ClusterName: name,
		},
	}
	return r
}

var testVirutalHostConfigs = map[string]v2.VirtualHost{
	"all":             v2.VirtualHost{Name: "all", Domains: []string{"*"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
	"wildcard-domain": v2.VirtualHost{Name: "wildcard-domain", Domains: []string{"*.sofa-mosn.test"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
	"domain":          v2.VirtualHost{Name: "domain", Domains: []string{"www.sofa-mosn.test"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
}
var testVirutalHostPortConfigs = map[string]v2.VirtualHost{
	"all": v2.VirtualHost{Name: "all", Domains: []string{"*:*"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
}

func TestNewRoutersSingle(t *testing.T) {
	// Single VirtualHost
	// Add VirtualHost type verify
	testCases := []struct {
		virtualHost v2.VirtualHost
		Expected    func(*routersImpl) bool
	}{
		{
			virtualHost: testVirutalHostConfigs["all"],
			Expected: func(rm *routersImpl) bool {
				return rm.defaultVirtualHostIndex != -1
			},
		},
		{
			virtualHost: testVirutalHostConfigs["wildcard-domain"],
			Expected: func(rm *routersImpl) bool {
				return len(rm.portWildcardVirtualHost) == 1
			},
		},
		{
			virtualHost: testVirutalHostConfigs["domain"],
			Expected: func(rm *routersImpl) bool {
				return len(rm.virtualHosts) == 1
			},
		},
	}
	for _, tc := range testCases {

		cfg := &v2.RouterConfiguration{
			VirtualHosts: []v2.VirtualHost{tc.virtualHost},
		}

		routers, err := NewRouters(cfg)
		if err != nil {
			t.Errorf("#%s : %v\n", tc.virtualHost.Name, err)
			continue
		}
		rm := routers.(*routersImpl)
		if !tc.Expected(rm) {
			t.Errorf("#%s : expected one virtualhost\n", tc.virtualHost.Name)
		}
	}
}

func TestNewRoutersGroup(t *testing.T) {
	//A group of VirtualHost
	var virtualhosts []v2.VirtualHost
	for _, vhConfig := range testVirutalHostConfigs {
		virtualhosts = append(virtualhosts, vhConfig)
	}
	cfg := &v2.RouterConfiguration{
		VirtualHosts: virtualhosts,
	}
	routers, err := NewRouters(cfg)
	if err != nil {
		t.Error(err)
		return
	}
	rm := routers.(*routersImpl)
	expected := rm.defaultVirtualHostIndex != -1 && len(rm.portWildcardVirtualHost) == 1 && len(rm.portWildcardVirtualHost) == 1
	if !expected {
		t.Error("create routematcher not match")
	}
}

func TestNewRoutersDuplicate(t *testing.T) {
	// two virtualhosts, both domain is "*", expected failed
	if _, err := NewRouters(&v2.RouterConfiguration{
		VirtualHosts: []v2.VirtualHost{testVirutalHostConfigs["all"], testVirutalHostConfigs["all"]},
	}); err == nil {
		t.Error("expected an error occur, but not")
	}
	// * and *:*
	if _, err := NewRouters(&v2.RouterConfiguration{
		VirtualHosts: []v2.VirtualHost{testVirutalHostConfigs["all"], testVirutalHostPortConfigs["all"]},
	}); err == nil {
		t.Error("expected an error occur, but not")
	}

	//two virtualhosts, both domain is "www.sofa-mosn.test", expected failed
	if _, err := NewRouters(&v2.RouterConfiguration{
		VirtualHosts: []v2.VirtualHost{testVirutalHostConfigs["domain"], testVirutalHostConfigs["domain"]},
	}); err == nil {
		t.Error("expected an error occur, but not")
	}
	// wildcard domain with same suffix, expected failed
	if _, err := NewRouters(&v2.RouterConfiguration{
		VirtualHosts: []v2.VirtualHost{testVirutalHostConfigs["wildcard-domain"], testVirutalHostConfigs["wildcard-domain"]},
	}); err == nil {
		t.Error("expected an error occur, but not")
	}
	// wildcard domain with different suffix:
	// *.test.com, *.test.net, *.test.com.cn
	// expected OK
	if _, err := NewRouters(&v2.RouterConfiguration{
		VirtualHosts: []v2.VirtualHost{
			v2.VirtualHost{Domains: []string{"*.test.com"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
			v2.VirtualHost{Domains: []string{"*.test.net"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
			v2.VirtualHost{Domains: []string{"*.test.com.cn"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
		},
	}); err != nil {
		t.Error("NewRouters with different wildcard domain failed")
	}

}

// match all
func TestDefaultMatch(t *testing.T) {
	cfg := &v2.RouterConfiguration{
		VirtualHosts: []v2.VirtualHost{
			testVirutalHostConfigs["all"],
		},
	}
	routers, err := NewRouters(cfg)
	if err != nil {
		t.Errorf("create router matcher failed %v\n", err)
		return
	}
	testCases := []string{
		"*.test.com",
		"*",
		"foo.com",
		"12345678",
	}
	ctx := variable.NewVariableContext(context.Background())
	for i, tc := range testCases {
		headers := protocol.CommonHeader(map[string]string{
			"service": "test",
		})
		variable.SetString(ctx, types.VarHost, tc)
		if routers.MatchRoute(ctx, headers) == nil {
			t.Errorf("#%d not matched\n", i)
		}
		if routers.MatchAllRoutes(ctx, headers) == nil {
			t.Errorf("#%d not matched\n", i)
		}
	}
}
func TestDomainMatch(t *testing.T) {
	cfg := &v2.RouterConfiguration{
		VirtualHosts: []v2.VirtualHost{
			testVirutalHostConfigs["domain"],
		},
	}
	routers, err := NewRouters(cfg)
	if err != nil {
		t.Errorf("create router matcher failed %v\n", err)
		return
	}
	ctx := variable.NewVariableContext(context.Background())
	headers := protocol.CommonHeader(map[string]string{
		"service": "test",
	})
	variable.SetString(ctx, types.VarHost, "www.sofa-mosn.test")
	if routers.MatchRoute(ctx, headers) == nil {
		t.Error("domain match failed")
	}
	if routers.MatchAllRoutes(ctx, headers) == nil {
		t.Error("domain match failed")
	}
	//not matched
	notMatched := []string{
		"*",
		"*www.sofa-mosn.test",
		"sofa-mosn.test",
		"www.sofa-mosn",
		"www.sofa-mosn.test1",
		"*.sofa-mosn.test",
	}
	for i, tc := range notMatched {
		headers := protocol.CommonHeader(map[string]string{
			"service": "test",
		})
		variable.SetString(ctx, types.VarHost, tc)
		if routers.MatchRoute(ctx, headers) != nil {
			t.Errorf("#%d expected not matched, but match a router", i)
		}
		if routers.MatchAllRoutes(ctx, headers) != nil {
			t.Errorf("#%d expected not matched, but match a router", i)
		}
	}
}
func TestWildcardMatch(t *testing.T) {
	testCases := []struct {
		wildcardDomain  string
		matchedDomain   []string
		unmatchedDomain []string
	}{
		{
			wildcardDomain:  "*.test.com",
			matchedDomain:   []string{"a.test.com", "a.test.test.com", "abc.test.com", "abc-def.test.com", "a.b.test.com", "*.test.com"},
			unmatchedDomain: []string{"a.test.net", "a-test.com", ".test.com", "*"},
		},
		{
			wildcardDomain:  "*.test.net",
			matchedDomain:   []string{"a.test.net"},
			unmatchedDomain: []string{"a.test.com"},
		},
		{
			wildcardDomain:  "*.test.com.cn",
			matchedDomain:   []string{"a.test.com.cn"},
			unmatchedDomain: []string{"a.test.com", "a.test.cn", "a.aaaa.com.cn"},
		},
		{
			wildcardDomain:  "*-bar.foo.com",
			matchedDomain:   []string{"a-bar.foo.com", "a.b-bar.foo.com", "a.-bar.foo.com"},
			unmatchedDomain: []string{"a.foo.com", "*.foo.com", "bar.foo.com", "-bar.test.com"},
		},
	}
	simpleRouter := newTestSimpleRouter("testRouter")
	ctx := variable.NewVariableContext(context.Background())

	for i, tc := range testCases {
		vh := v2.VirtualHost{
			Domains: []string{tc.wildcardDomain},
			Routers: []v2.Router{simpleRouter},
		}
		cfg := &v2.RouterConfiguration{
			VirtualHosts: []v2.VirtualHost{vh},
		}
		routers, err := NewRouters(cfg)
		if err != nil {
			t.Errorf("#%d create routers failed: %v\n", i, err)
			continue
		}
		for _, match := range tc.matchedDomain {
			headers := protocol.CommonHeader(map[string]string{
				"service": "test",
			})
			variable.SetString(ctx, types.VarHost, match)
			if routers.MatchRoute(ctx, headers) == nil {
				t.Errorf("%s expected matched: #%d, but return nil\n", match, i)
			}
			if routers.MatchAllRoutes(ctx, headers) == nil {
				t.Errorf("%s expected matched: #%d, but return nil\n", match, i)
			}
		}
		for _, unmatch := range tc.unmatchedDomain {
			headers := protocol.CommonHeader(map[string]string{
				"service": "test",
			})
			variable.SetString(ctx, types.VarHost, unmatch)
			if routers.MatchRoute(ctx, headers) != nil {
				t.Errorf("%s expected unmatched: #%d, but matched\n", unmatch, i)
			}
			if routers.MatchAllRoutes(ctx, headers) != nil {
				t.Errorf("%s expected unmatched: #%d, but matched\n", unmatch, i)
			}
		}
	}

}

func TestWildcardLongestSuffixMatch(t *testing.T) {
	virtualHosts := []v2.VirtualHost{
		v2.VirtualHost{Domains: []string{"f-bar.baz.com"}, Routers: []v2.Router{newTestSimpleRouter("domain")}},
		v2.VirtualHost{Domains: []string{"*.baz.com"}, Routers: []v2.Router{newTestSimpleRouter("short")}},
		v2.VirtualHost{Domains: []string{"*-bar.baz.com"}, Routers: []v2.Router{newTestSimpleRouter("long")}},
		v2.VirtualHost{Domains: []string{"*.foo.com"}, Routers: []v2.Router{newTestSimpleRouter("foo")}},
	}
	cfg := &v2.RouterConfiguration{
		VirtualHosts: virtualHosts,
	}
	routers, err := NewRouters(cfg)
	if err != nil {
		t.Error(err)
		return
	}
	testCases := []struct {
		Domain        string
		ExpectedRoute string
	}{
		{Domain: "f-bar.baz.com", ExpectedRoute: "domain"},
		{Domain: "foo-bar.baz.com", ExpectedRoute: "long"},
		{Domain: "foo.baz.com", ExpectedRoute: "short"},
		{Domain: "foo.foo.com", ExpectedRoute: "foo"},
	}
	ctx := variable.NewVariableContext(context.Background())
	for _, tc := range testCases {
		variable.SetString(ctx, types.VarHost, tc.Domain)
		route := routers.MatchRoute(ctx, protocol.CommonHeader(map[string]string{
			"service": "test",
		}))
		if route == nil {
			t.Errorf("%s match failed\n", tc.Domain)
			continue
		}
		if route.RouteRule().ClusterName(context.TODO()) != tc.ExpectedRoute {
			t.Errorf("%s expected match %s, but got %s\n", tc.Domain, tc.ExpectedRoute, route.RouteRule().ClusterName(context.TODO()))
		}
	}
}

func TestVirtulHostWithPortMatch(t *testing.T) {
	virtualHosts := []v2.VirtualHost{
		{Domains: []string{"www.test.com"}, Routers: []v2.Router{newTestSimpleRouter("0")}},
		{Domains: []string{"www.test.com:8080"}, Routers: []v2.Router{newTestSimpleRouter("1")}},
		{Domains: []string{"www.test.com:*"}, Routers: []v2.Router{newTestSimpleRouter("2")}},
		{Domains: []string{"*.test.com:30888"}, Routers: []v2.Router{newTestSimpleRouter("3")}},
		{Domains: []string{"*.com:30888"}, Routers: []v2.Router{newTestSimpleRouter("4")}},
		{Domains: []string{"*.com:*"}, Routers: []v2.Router{newTestSimpleRouter("5")}},
		{Domains: []string{"*.com"}, Routers: []v2.Router{newTestSimpleRouter("5")}},
		{Domains: []string{"*:*"}, Routers: []v2.Router{newTestSimpleRouter("6")}},
		{Domains: []string{"*:80"}, Routers: []v2.Router{newTestSimpleRouter("7")}},
	}

	cfg := &v2.RouterConfiguration{
		VirtualHosts: virtualHosts,
	}
	routers, err := NewRouters(cfg)
	if err != nil {
		t.Error(err)
		return
	}
	testCases := []struct {
		Domain        string
		ExpectedRoute string
	}{
		{Domain: "www.test.com", ExpectedRoute: "0"},
		{Domain: "www.test.com:8080", ExpectedRoute: "1"},
		{Domain: "www.test.com:80", ExpectedRoute: "2"},
		{Domain: "www.test.com:30888", ExpectedRoute: "2"},
		{Domain: "hello.test.com:30888", ExpectedRoute: "3"},
		{Domain: "hello.com:30888", ExpectedRoute: "4"},
		{Domain: "hello.com:30777", ExpectedRoute: "5"},
		{Domain: "hello.cn:30777", ExpectedRoute: "6"},
		{Domain: "hello.cn", ExpectedRoute: "6"},
		{Domain: "hello.cn:80", ExpectedRoute: "7"},
	}
	ctx := variable.NewVariableContext(context.Background())
	for _, tc := range testCases {
		variable.SetString(ctx, types.VarHost, tc.Domain)
		route := routers.MatchRoute(ctx, protocol.CommonHeader(map[string]string{
			"service": "test",
		}))
		if route == nil {
			t.Errorf("%s match failed\n", tc.Domain)
			continue
		}
		if route.RouteRule().ClusterName(context.TODO()) != tc.ExpectedRoute {
			t.Errorf("%s expected match %s, but got %s\n", tc.Domain, tc.ExpectedRoute, route.RouteRule().ClusterName(context.TODO()))
		}
	}
}

func TestAddRouter(t *testing.T) {
	// 1. no routes
	cfg := &v2.RouterConfiguration{
		VirtualHosts: []v2.VirtualHost{
			v2.VirtualHost{
				Domains: []string{"www.test.com"},
			},
			v2.VirtualHost{
				Domains: []string{"*"},
			},
		},
	}
	rm, err := NewRouters(cfg)
	if err != nil {
		t.Fatal("create routers failed")
	}
	matcher := rm.(*routersImpl)
	vh := matcher.virtualHosts[0].(*VirtualHostImpl)
	defaultVh := matcher.virtualHosts[1].(*VirtualHostImpl)
	if len(vh.routes) != 0 || len(defaultVh.routes) != 0 {
		t.Fatal("expected a no routes matcher")
	}
	route := newTestSimpleRouter("test_add")
	if index := rm.AddRoute("www.test.com", &route); index == -1 {
		t.Fatal("add route failed")
	}
	if len(vh.routes) != 1 || len(defaultVh.routes) != 0 {
		t.Fatal("expected add a new route")
	}
	// add into default virtual host (match any thing)
	if index := rm.AddRoute("", &route); index == -1 {
		t.Fatal("add route failed")
	}
	if len(vh.routes) != 1 || len(defaultVh.routes) != 1 {
		t.Fatal("expected add a new route into default")
	}
}

func TestRemoveAllRoutes(t *testing.T) {
	// init
	cfg := &v2.RouterConfiguration{
		VirtualHosts: []v2.VirtualHost{
			v2.VirtualHost{
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
			v2.VirtualHost{
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
	rm, err := NewRouters(cfg)
	if err != nil {
		t.Fatal("create routers failed")
	}
	matcher := rm.(*routersImpl)
	vh := matcher.virtualHosts[0].(*VirtualHostImpl)
	defaultVh := matcher.virtualHosts[1].(*VirtualHostImpl)
	if len(vh.routes) != 1 || len(defaultVh.routes) != 1 {
		t.Fatal("expected exists routes matcher")
	}
	if index := rm.RemoveAllRoutes("www.test.com"); index == -1 {
		t.Fatal("remove route failed")
	}
	if len(vh.routes) != 0 || len(defaultVh.routes) != 1 {
		t.Fatal("expected remove route")
	}
	// remove default virtual host
	if index := rm.RemoveAllRoutes(""); index == -1 {
		t.Fatal("expected remove route")
	}
	if len(vh.routes) != 0 || len(defaultVh.routes) != 0 {
		t.Fatal("expected add a new route into default")
	}
}

func TestInvalidConfig(t *testing.T) {
	// nil config
	if _, err := NewRouters(nil); err == nil {
		t.Errorf("nil config should return an error")
	}
	// no virtual host
	cfg := &v2.RouterConfiguration{}
	if _, err := NewRouters(cfg); err == nil {
		t.Errorf("config should have at least one virtual host")
	}
	// duplicate virtual host
	cfg.VirtualHosts = []v2.VirtualHost{
		{
			Domains: []string{"*"},
		},
		{
			Domains: []string{"*"},
		},
	}
	if _, err := NewRouters(cfg); err == nil {
		t.Errorf("config should not have duplicate virtual host name")
	}
}

func TestMain(m *testing.M) {
	log.InitDefaultLogger("", log.DEBUG)
	os.Exit(m.Run())
}
